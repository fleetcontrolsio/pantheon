package pantheon

import (
	"net/http"
	"time"

	"github.com/fleetcontrolsio/pantheon/pkg/hashring"
)

type Options struct {
	prefix string
	// The name of the cluster
	// This used as a prefix for cluster keys in redis
	name string
	// The interval at which the cluster will send heartbeat messages to other nodes
	hearbeatInterval time.Duration
	// The timeout for heartbeat messages
	heartbeatTimeout time.Duration
	// heartbeatConcurrency; the number of concurrent heartbeat checks
	heartbeatConcurrency int
	// The maximum number of failed heartbeat requests before a node is considered dead
	heartbeatMaxFailures int
	// The hostname of the redis server/cluster
	redisHost string
	// The port of the redis server/cluster
	redisPort int
	// The password for the redis server/cluster
	redisPassword string
	// The database to use in the redis server/cluster
	redisDB int
	// The maximum number of retries before giving up
	redisMaxRetries int
	// The retry backoff interval
	redisRetryBackoff time.Duration
	// The http client for heartbeat requests
	httpClient *http.Client
	// hashRing: the hash ring for the cluster
	hashRing hashring.Ring
	// hashringReplicaCount: number of virtual nodes per physical node in the hash ring
	hashringReplicaCount int
}

// NewOptions creates a new Options instance with default values
// The default values are:
// - prefix: "pantheon"
// - name: "my-cluster"
// - hearbeatInterval: 30 seconds
// - heartbeatTimeout: 30 seconds
// - heartbeatConcurrency: 2
// - heartbeatMaxFailures: 3
// - redisHost: "localhost"
// - redisPort: 6379
// - redisDB: 0
// - redisMaxRetries: 5
// - redisRetryBackoff: 20 seconds
// - hashringReplicaCount: 10
// - httpClient: nil
// - hashRing: nil
func NewOptions() *Options {
	return &Options{
		prefix:               "pantheon",
		name:                 "my-cluster",
		hearbeatInterval:     30 * time.Second,
		heartbeatConcurrency: 2,
		heartbeatMaxFailures: 5,
		redisHost:            "localhost",
		redisPort:            6379,
		redisDB:              0,
		redisMaxRetries:      5,
		redisRetryBackoff:    20 * time.Second,
		hashringReplicaCount: 10, // Default to 10 virtual nodes per physical node
	}
}

func (o *Options) WithPrefix(prefix string) *Options {
	o.prefix = prefix
	return o
}

func (o *Options) WithName(name string) *Options {
	o.name = name
	return o
}

func (o *Options) WithHeartbeatInterval(interval time.Duration) *Options {
	o.hearbeatInterval = interval
	return o
}

func (o *Options) WithHeartbeatTimeout(timeout time.Duration) *Options {
	o.heartbeatTimeout = timeout
	return o
}

func (o *Options) WithHearbeatConcurrency(concurrency int) *Options {
	o.heartbeatConcurrency = concurrency
	return o
}

func (o *Options) WithHeartbeatMaxFailures(maxFailures int) *Options {
	o.heartbeatMaxFailures = maxFailures
	return o
}

func (o *Options) WithRedisHost(host string) *Options {
	o.redisHost = host
	return o
}

func (o *Options) WithRedisPort(port int) *Options {
	o.redisPort = port
	return o
}

func (o *Options) WithRedisPassword(password string) *Options {
	o.redisPassword = password
	return o
}

func (o *Options) WithRedisDB(db int) *Options {
	o.redisDB = db
	return o
}

func (o *Options) WithRedisMaxRetries(retries int) *Options {
	o.redisMaxRetries = retries
	return o
}

func (o *Options) WithRedisRetryBackoff(interval time.Duration) *Options {
	o.redisRetryBackoff = interval
	return o
}

func (o *Options) WithHTTPClient(client *http.Client) *Options {
	o.httpClient = client
	return o
}

func (o *Options) WithHashRing(ring hashring.Ring) *Options {
	o.hashRing = ring
	return o
}

func (o *Options) WithHashRingReplicaCount(count int) *Options {
	o.hashringReplicaCount = count
	return o
}

func (o *Options) Validate() error {
	if o.prefix == "" {
		return ErrInvalidPrefix
	}

	if o.name == "" {
		return ErrInvalidName
	}

	if o.hearbeatInterval <= 0 {
		return ErrInvalidHeartbeatInterval
	}

	if o.heartbeatTimeout <= 0 {
		return ErrInvalidHeartbeatTimeout
	}

	if o.heartbeatConcurrency <= 0 {
		return ErrInvalidHeartbeatConcurrency
	}

	if o.heartbeatMaxFailures <= 0 {
		return ErrInvalidHeartbeatMaxFailures
	}

	if o.redisHost == "" {
		return ErrInvalidRedisHost
	}

	if o.redisPort <= 0 {
		return ErrInvalidRedisPort
	}

	if o.redisDB < 0 {
		return ErrInvalidRedisDB
	}

	if o.redisMaxRetries < 0 {
		return ErrInvalidRedisMaxRetries
	}

	if o.redisRetryBackoff <= 0 {
		return ErrInvalidRedisRetryBackoff
	}

	if o.httpClient == nil {
		return ErrInvalidHTTPClient
	}

	if o.hashRing == nil {
		return ErrInvalidHashRing
	}

	return nil
}
