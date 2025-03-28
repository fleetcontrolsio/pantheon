package pantheon

import (
	"net/http"
	"time"
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
}

func NewOptions() *Options {
	return &Options{
		prefix:               "pantheon",
		name:                 "mypantheon",
		hearbeatInterval:     30 * time.Second,
		heartbeatConcurrency: 2,
		redisHost:            "localhost",
		redisPort:            6379,
		redisDB:              0,
		redisMaxRetries:      5,
		redisRetryBackoff:    20 * time.Second,
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

	return nil
}
