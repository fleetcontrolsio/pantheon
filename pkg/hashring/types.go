package hashring

// Ring defines the interface for a consistent hash ring
type Ring interface {
	// AddNode adds a new node to the hash ring
	AddNode(node *Node) error

	// RemoveNode removes a node from the hash ring
	RemoveNode(nodeID string) error

	// GetNode returns the node responsible for the given key
	GetNode(key string) (*Node, error)

	// GetNodes returns all nodes in the hash ring
	GetNodes() []*Node

	// GetNodeCount returns the number of nodes in the hash ring
	GetNodeCount() int

	// UpdateNodeStatus updates a node's status
	UpdateNodeStatus(nodeID string, status NodeStatus) error
}
