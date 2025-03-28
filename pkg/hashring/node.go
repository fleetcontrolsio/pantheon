package hashring

// Status represents the current state of a node
type NodeStatus string

const (
	// StatusActive indicates the node is operational and available
	NodeStatusActive NodeStatus = "active"

	// StatusInactive indicates the node is not operational
	NodeStatusInactive NodeStatus = "inactive"

	// StatusDraining indicates the node is preparing to be removed
	NodeStatusDraining NodeStatus = "draining"
)

// Node represents a physical node in the system
type Node struct {
	// ID is the unique identifier for this node
	ID string

	// Address is the network address of the node
	Address string

	// Status indicates the current operational status
	Status NodeStatus

	// LastHeartbeat is the Unix timestamp of the last heartbeat received
	LastHeartbeat int64
}

// NewNode creates a new node with the given ID and address
func NewNode(id, address string) *Node {
	return &Node{
		ID:      id,
		Address: address,
		Status:  NodeStatusActive,
	}
}

// IsAvailable returns true if the node is available to handle requests
func (n *Node) IsAvailable() bool {
	return n.Status == NodeStatusActive
}

// SetStatus updates the node's status
func (n *Node) SetStatus(status NodeStatus) {
	n.Status = status
}

// UpdateHeartbeat updates the last heartbeat timestamp
func (n *Node) UpdateHeartbeat(timestamp int64) {
	n.LastHeartbeat = timestamp
}
