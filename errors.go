package pantheon

import (
	"errors"
	"fmt"
)

// Option validation errors
var ErrInvalidPrefix = errors.New("prefix is required")

var ErrInvalidName = errors.New("name is required")

var ErrInvalidHeartbeatInterval = errors.New("heartbeat interval must be greater than 0")

var ErrInvalidHeartbeatTimeout = errors.New("heartbeat timeout must be greater than 0")

var ErrInvalidHeartbeatConcurrency = errors.New("heartbeat concurrency must be greater than 0")

var ErrInvalidRedisHost = errors.New("redis host is required")

var ErrInvalidRedisPort = errors.New("redis port is required")

var ErrInvalidRedisDB = errors.New("redis db must be greater than or equal to 0")

var ErrInvalidRedisMaxRetries = errors.New("redis max retries must be greater than or equal to 0")

var ErrInvalidRedisRetryBackoff = errors.New("redis retry backoff must be greater than 0")

var ErrInvalidHTTPClient = errors.New("http client is required")

type ErrNodePropertyNotFound struct {
	property string
}

func NewErrNodePropertyNotFound(property string) *ErrNodePropertyNotFound {
	return &ErrNodePropertyNotFound{property: property}
}

func (e *ErrNodePropertyNotFound) Error() string {
	return fmt.Sprintf("%s not found for node", e.property)
}
