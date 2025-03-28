package pantheon

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/redis/go-redis/v9"
)

type RedisClient interface {
	Ping(ctx context.Context) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	HSet(ctx context.Context, key string, fields ...interface{}) *redis.IntCmd
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	HIncrBy(ctx context.Context, key, field string, incr int64) *redis.IntCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
}

type RedisClientOptions struct {
	// The hostname of the redis server/cluster
	Host string
	// The port of the redis server/cluster
	Port int
	// The password for the redis server/cluster
	Password string
	// The database to use in the redis server/cluster
	DB int
	// The maximum number of retries before giving up
	MaxRetries int
	// The maximum time to wait before giving up
	RetryBackOffLimit time.Duration
}

// NewRedisClient creates a new redis client
func NewRedisClient(ctx context.Context, opts *RedisClientOptions) (RedisClient, error) {
	var lastError error = nil
	connectionAttempts := 0

	connectionRetryBackoff := backoff.NewExponentialBackOff()
	connectionRetryBackoff.MaxElapsedTime = opts.RetryBackOffLimit

	redisAddr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)

	clientOpts := &redis.Options{
		Addr: redisAddr,
	}

	if opts.Password != "" {
		clientOpts.Password = opts.Password
	}

	client := redis.NewClient(clientOpts)

	// Start the connection loop
	for {
		if err := client.Ping(ctx).Err(); err != nil {
			lastError = err
			connectionAttempts++
			// nextRetryAt := time.Now().Add(connectionRetryBackoff.NextBackOff())
			if connectionAttempts > opts.MaxRetries {
				lastError = fmt.Errorf("failed to connect to redis server after %d attempts", connectionAttempts)
				break
			}
			time.Sleep(connectionRetryBackoff.NextBackOff())
		} else {
			connectionRetryBackoff.Reset()
			break
		}
	}

	if lastError != nil {
		return nil, lastError
	}

	return client, nil
}
