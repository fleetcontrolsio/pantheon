package hashring

import "errors"

// ErrNoNodes is returned when attempting to get a node from an empty ring
var ErrNoNodes = errors.New("no nodes available in the hash ring")

// ErrNodeExists is returned when attempting to add a node that already exists
var ErrNodeExists = errors.New("node already exists in the hash ring")

// ErrNodeNotFound is returned when attempting to operate on a node that doesn't exist
var ErrNodeNotFound = errors.New("node not found in the hash ring")
