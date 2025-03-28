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
		pool.Go(func() {
			ctx, cancel := context.WithTimeout(ctx, c.heartbeatTimeout)
			defer cancel()
			// TODO: Handle the context deadline exceeded error and increment the node's failure count in redis
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
}
