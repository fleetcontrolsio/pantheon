package pantheon

import (
	"context"
	"net/http"
	"time"
)

type Pantheon struct {
	// ctx: the context for the cluster
	ctx context.Context
	// storage; the storage for the cluster
	storage *Storage
	// http: http client for heartbeat requests
	http *http.Client
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

// Distribute distributes keys to the nodes in the cluster
// This should be called after a nodes has joined or left the cluster to
// rebalance the keys to available nodes
func (c *Pantheon) Distribute(keys []string) error {
	return nil
}

// Join adds a node to the cluster
// This should be called when a new node is starting up
func (c *Pantheon) Join(op *JoinOp) error {
	return nil
}

// Leave removes a node from the cluster
// This should called when a node is shutting down
func (c *Pantheon) Leave(id string) error {
	return nil
}
