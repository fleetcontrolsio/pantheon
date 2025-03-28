package pantheon

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetcontrolsio/pantheon/pkg/hashring"
)

type Pantheon struct {
	// ctx: the context for the cluster
	ctx context.Context
	// storage; the storage for the cluster
	storage *Storage
	// http: http client for heartbeat requests
	http *http.Client
	// hashRing: the hash ring for the cluster
	hashRing hashring.Ring
	// name; the name of the cluster
	name string
	// hearbeat; the interval at which the cluster will send heartbeat messages to other nodes
	hearbeat *time.Ticker
	// heartbeatConcurrency; the number of concurrent heartbeat checks
	heartbeatConcurrency int
	// heartbeatTimeout; the timeout for heartbeat checks
	// if a node does not respond within this time, it is considered dead
	heartbeatTimeout time.Duration
	// heartbeatMaxFailures; the maximum number of failed heartbeat requests before a node is considered dead
	heartbeatMaxFailures int
	// heartbeatEventCh; a channel to send heartbeat events
	heartbeatEventCh chan HearbeatEvent
	// eventsCh; a channel to send cluster events
	EventsCh chan PantheonEvent
	// started; a flag to indicate if the cluster has been started
	started bool
}

type JoinOp struct {
	// ID; the name of the node joining the cluster
	ID string
	// Address; the address of the node joining the cluster
	Address string
	// Port; the port of the node joining the cluster
	Port int
	// Path: the path on the node to make the heartbeat request to
	Path string
}

// New create a new Pantheon instance
func New(ctx context.Context, options *Options) (*Pantheon, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	redisClient, err := NewRedisClient(ctx, &RedisClientOptions{
		Host:              options.redisHost,
		Port:              options.redisPort,
		Password:          options.redisPassword,
		DB:                options.redisDB,
		MaxRetries:        options.redisMaxRetries,
		RetryBackOffLimit: options.redisRetryBackoff,
	})
	if err != nil {
		return nil, err
	}

	storage := NewStorage(options.prefix, options.name, redisClient)

	// Create a hash ring if one is not provided
	var ring hashring.Ring
	if options.hashRing != nil {
		ring = options.hashRing
	} else {
		ring = hashring.NewHashRing(options.hashringReplicaCount)
	}

	return &Pantheon{
		ctx:                  ctx,
		name:                 options.name,
		storage:              storage,
		http:                 options.httpClient,
		hearbeat:             time.NewTicker(options.hearbeatInterval),
		heartbeatTimeout:     options.heartbeatTimeout,
		heartbeatConcurrency: options.heartbeatConcurrency,
		heartbeatMaxFailures: options.heartbeatMaxFailures,
		heartbeatEventCh:     make(chan HearbeatEvent),
		hashRing:             ring,
		EventsCh:             make(chan PantheonEvent),
		started:              false,
	}, nil
}

// Start starts the cluster
// this should be called before adding nodes to the cluster
func (c *Pantheon) Start() error {
	if c.started {
		return nil
	}
	c.started = true

	// handle the heartbeat events
	go func() {
		for {
			select {
			case event := <-c.heartbeatEventCh:
				c.handleHeartbeatEvent(event)
			case <-c.ctx.Done():
				// Close the heartbeat event channel
				close(c.heartbeatEventCh)
				return
			}
		}
	}()

	// start the heartbeat loop
	go func() {
		for {
			select {
			case <-c.hearbeat.C:
				ctx, cancel := context.WithTimeout(c.ctx, c.heartbeatTimeout)
				defer cancel()

				c.performHeartbeat(ctx)
			case <-c.ctx.Done():
				c.hearbeat.Stop()
				return
			}
		}
	}()

	return nil
}

// Destroy stops the cluster
// this should be called when the cluster is no longer needed
func (c *Pantheon) Destroy() error {
	if !c.started {
		return nil
	}

	c.started = false

	return nil
}

// Join adds a node to the cluster
// This should be called when a new node is starting up
func (c *Pantheon) Join(op *JoinOp) error {
	if !c.started {
		return fmt.Errorf("cluster not started")
	}

	// Use the Pantheon context instead of creating a new one
	// Check if the node already exists
	node, err := c.storage.GetNode(c.ctx, op.ID)
	if err != nil {
		return err
	}

	if node != nil {
		return fmt.Errorf("node %s already exists", op.ID)
	}

	// Add the node to the cluster
	addr := fmt.Sprintf("%s:%d", op.Address, op.Port)
	err = c.storage.AddNode(c.ctx, op.ID, op.Address, op.Path, op.Port)
	if err != nil {
		return err
	}
	// Add the node to the hash ring
	err = c.hashRing.AddNode(&hashring.Node{
		ID:      op.ID,
		Address: addr,
		Status:  hashring.NodeStatusActive,
	})
	if err != nil {
		return err
	}

	// Send a joined event
	if c.EventsCh != nil {
		c.EventsCh <- PantheonEvent{
			Event:  "joined",
			NodeID: op.ID,
		}
	}

	// Immediately ping the node to check its health
	go func() {
		if err := c.PingNode(op.ID); err != nil {
			fmt.Printf("error pinging new node %s: %s\n", op.ID, err)
		}
	}()

	fmt.Printf("Node %s (%s) joined the cluster\n", op.ID, addr)
	return nil
}

// Leave removes a node from the cluster
// This should called when a node is shutting down
func (c *Pantheon) Leave(id string) error {
	if !c.started {
		return fmt.Errorf("cluster not started")
	}

	// Use the Pantheon context instead of creating a new one
	// Check if the node exists
	node, err := c.storage.GetNode(c.ctx, id)
	if err != nil {
		return err
	}

	if node == nil {
		return fmt.Errorf("node %s not found", id)
	}

	// Remove the node from the cluster
	err = c.storage.RemoveNode(c.ctx, id)
	if err != nil {
		return err
	}

	// Remove the node from the hash ring
	err = c.hashRing.RemoveNode(id)
	if err != nil {
		return err
	}

	// Send a left event
	if c.EventsCh != nil {
		c.EventsCh <- PantheonEvent{
			Event:  "left",
			NodeID: id,
		}
	}

	fmt.Printf("Node %s left the cluster\n", id)
	return nil
}
