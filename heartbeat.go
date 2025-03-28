package pantheon

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sourcegraph/conc/pool"
)

type HearbeatEvent struct {
	NodeID string
	// Event; the event that occurred
	Event string
	// Error; the error that occurred
	Error error
}

func (c *Pantheon) performHeartbeat(ctx context.Context) {
	// get the nodes
	nodes, err := c.storage.GetNodes(ctx)
	if err != nil {
		return
	}

	pool := pool.New().WithMaxGoroutines(c.heartbeatConcurrency)

	for _, node := range nodes {
		node := node // Create a local copy for the goroutine
		pool.Go(func() {
			// Increment heartbeat count
			if err := c.storage.IncrementHeartbeats(ctx, node.ID); err != nil {
				fmt.Printf("error incrementing heartbeat count: %s\n", err)
			}

			c.performHearbeatRequest(ctx, &node)
		})
	}

	pool.Wait()
}

func (c *Pantheon) performHearbeatRequest(ctx context.Context, node *Member) {
	url := fmt.Sprintf("%s/%s", node.Address, node.Path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// TODO: Use a logger
		fmt.Printf("error creating request: %s\n", err)
		return
	}
	// Use the parent context so that the request is cancelled if the parent context is cancelled
	resp, err := c.http.Do(req.WithContext(ctx))
	if err != nil {
		fmt.Printf("error making request: %s\n", err)
		c.heartbeatEventCh <- HearbeatEvent{
			NodeID: node.ID,
			Event:  "failure",
			Error:  fmt.Errorf("hearbeat request to %s failed: %s", url, err.Error()),
		}
		return
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("unexpected status code: %d\n", resp.StatusCode)
		c.heartbeatEventCh <- HearbeatEvent{
			NodeID: node.ID,
			Event:  "failure",
			Error:  fmt.Errorf("hearbeat request to %s failed with status code %d", url, resp.StatusCode),
		}

		return
	}

	c.heartbeatEventCh <- HearbeatEvent{
		NodeID: node.ID,
		Event:  "success",
		Error:  nil,
	}
}

// handleHeartbeatEvent handles the heartbeat events
func (c *Pantheon) handleHeartbeatEvent(event HearbeatEvent) {
	// Use the context from the Pantheon struct instead of creating a new one
	// Get the current node data
	node, err := c.storage.GetNode(c.ctx, event.NodeID)
	if err != nil {
		fmt.Printf("error getting node %s: %s\n", event.NodeID, err)
		return
	}

	if node == nil {
		fmt.Printf("node %s not found\n", event.NodeID)
		return
	}

	if event.Event == "success" {
		// Update the node's last heartbeat
		if err := c.storage.UpdateNodeHeartbeat(c.ctx, event.NodeID); err != nil {
			fmt.Printf("error updating heartbeat: %s\n", err)
			return
		}

		// If the node was previously dead or suspect, mark it as alive
		if node.State != "alive" {
			if err := c.storage.UpdateNodeState(c.ctx, event.NodeID, "alive"); err != nil {
				fmt.Printf("error updating node state: %s\n", err)
				return
			}
			
			// Update the node status in the hash ring
			if c.hashRing != nil {
				err = c.hashRing.UpdateNodeStatus(event.NodeID, hashring.NodeStatusActive)
				if err != nil && err != hashring.ErrNodeNotFound {
					fmt.Printf("error updating node status in hash ring: %s\n", err)
				}
			}

			// Send a node revived event
			if c.EventsCh != nil {
				c.EventsCh <- PantheonEvent{
					Event:  "revived",
					NodeID: event.NodeID,
				}
			}
		}
	} else if event.Event == "failure" {
		// Increment the failure count
		if err := c.storage.IncrementHeartbeatFailures(c.ctx, event.NodeID); err != nil {
			fmt.Printf("error incrementing failure count: %s\n", err)
			return
		}

		// Check if the node has exceeded the maximum failure count
		failures, err := getHeartbeatFailureCount(node.HeartbeatFailures)
		if err != nil {
			fmt.Printf("error parsing failure count: %s\n", err)
			return
		}

		if failures >= c.heartbeatMaxFailures {
			// Mark the node as dead
			if node.State != "dead" {
				if err := c.storage.UpdateNodeState(c.ctx, event.NodeID, "dead"); err != nil {
					fmt.Printf("error updating node state: %s\n", err)
					return
				}
				
				// Update the node status in the hash ring
				if c.hashRing != nil {
					err = c.hashRing.UpdateNodeStatus(event.NodeID, hashring.NodeStatusInactive)
					if err != nil && err != hashring.ErrNodeNotFound {
						fmt.Printf("error updating node status in hash ring: %s\n", err)
					}
				}

				// Send a node dead event
				if c.EventsCh != nil {
					c.EventsCh <- PantheonEvent{
						Event:  "died",
						NodeID: event.NodeID,
					}
				}
				
				// Trigger rebalancing after a node is marked dead
				go func() {
					// Get all keys assigned to this node
					nodeKeysKey := c.storage.makeKey("nodekeys", event.NodeID)
					keys, err := c.storage.redis.SMembers(c.ctx, nodeKeysKey).Result()
					if err != nil {
						fmt.Printf("error getting keys for dead node: %s\n", err)
						return
					}
					
					if len(keys) > 0 {
						fmt.Printf("Redistributing %d keys from dead node %s\n", len(keys), event.NodeID)
						if err := c.Distribute(keys); err != nil {
							fmt.Printf("error redistributing keys: %s\n", err)
						}
					}
				}()
			}
		} else if node.State == "alive" {
			// Mark the node as suspect
			if err := c.storage.UpdateNodeState(c.ctx, event.NodeID, "suspect"); err != nil {
				fmt.Printf("error updating node state: %s\n", err)
				return
			}
		}
	}
}

// getHeartbeatFailureCount parses the heartbeat failure count from a string
func getHeartbeatFailureCount(count string) (int, error) {
	var value int
	_, err := fmt.Sscanf(count, "%d", &value)
	if err != nil {
		return 0, err
	}
	return value, nil
}

// GetNodeHealth returns the health status of a node
func (c *Pantheon) GetNodeHealth(nodeID string) (MemberState, error) {
	// Use the context from the Pantheon struct
	node, err := c.storage.GetNode(c.ctx, nodeID)
	if err != nil {
		return "", err
	}

	if node == nil {
		return "", fmt.Errorf("node %s not found", nodeID)
	}

	return node.State, nil
}

// ResetNodeFailures resets the heartbeat failure count for a node
func (c *Pantheon) ResetNodeFailures(nodeID string) error {
	return c.storage.ResetHeartbeatFailures(c.ctx, nodeID)
}

// PingNode forces an immediate heartbeat check for a node
func (c *Pantheon) PingNode(nodeID string) error {
	// Use the context from the Pantheon struct
	node, err := c.storage.GetNode(c.ctx, nodeID)
	if err != nil {
		return err
	}

	if node == nil {
		return fmt.Errorf("node %s not found", nodeID)
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.heartbeatTimeout)
	defer cancel()

	c.performHearbeatRequest(ctx, node)

	return nil
}
