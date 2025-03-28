package hashring

import (
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
	"sync"
)

// HashRing implements a consistent hash ring
type HashRing struct {
	nodes        map[string]*Node  // Map of node ID to node
	virtualNodes map[uint32]string // Map of virtual node hash to node ID
	sortedHashes []uint32          // Sorted list of virtual node hashes
	replicaCount int               // Number of virtual nodes per physical node
	mu           sync.RWMutex      // Protects access to the hash ring
}

// NewHashRing creates a new consistent hash ring
func NewHashRing(replicaCount int) *HashRing {
	if replicaCount <= 0 {
		replicaCount = 10 // Default to 10 replicas if invalid count provided
	}

	return &HashRing{
		nodes:        make(map[string]*Node),
		virtualNodes: make(map[uint32]string),
		sortedHashes: make([]uint32, 0),
		replicaCount: replicaCount,
	}
}

// AddNode adds a node to the hash ring
func (h *HashRing) AddNode(n *Node) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if n == nil {
		return errors.New("cannot add nil node")
	}

	if n.ID == "" {
		return errors.New("node ID cannot be empty")
	}

	if _, exists := h.nodes[n.ID]; exists {
		return ErrNodeExists
	}

	// Add the node to our nodes map
	h.nodes[n.ID] = n

	// Add virtual nodes
	for i := 0; i < h.replicaCount; i++ {
		virtualNodeKey := fmt.Sprintf("%s:%d", n.ID, i)
		hash := crc32.ChecksumIEEE([]byte(virtualNodeKey))
		h.virtualNodes[hash] = n.ID
		h.sortedHashes = append(h.sortedHashes, hash)
	}

	// Resort hashes
	sort.Slice(h.sortedHashes, func(i, j int) bool {
		return h.sortedHashes[i] < h.sortedHashes[j]
	})

	return nil
}

// RemoveNode removes a node from the hash ring
func (h *HashRing) RemoveNode(nodeID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	node, exists := h.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	// Remove the node from our nodes map
	delete(h.nodes, nodeID)

	// Remove virtual nodes
	newSortedHashes := make([]uint32, 0, len(h.sortedHashes)-h.replicaCount)
	for i := 0; i < h.replicaCount; i++ {
		virtualNodeKey := fmt.Sprintf("%s:%d", node.ID, i)
		hash := crc32.ChecksumIEEE([]byte(virtualNodeKey))
		delete(h.virtualNodes, hash)

		// Rebuild the sorted hashes array excluding this node's hashes
		for _, h := range h.sortedHashes {
			if h != hash {
				newSortedHashes = append(newSortedHashes, h)
			}
		}
	}

	h.sortedHashes = newSortedHashes
	return nil
}

// GetNode returns the node responsible for the given key
func (h *HashRing) GetNode(key string) (*Node, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.nodes) == 0 {
		return nil, ErrNoNodes
	}

	// Get the hash of the key
	hash := crc32.ChecksumIEEE([]byte(key))

	// Find the first virtual node with hash >= key hash
	idx := sort.Search(len(h.sortedHashes), func(i int) bool {
		return h.sortedHashes[i] >= hash
	})

	// If we reached the end, wrap around to the first virtual node
	if idx >= len(h.sortedHashes) {
		idx = 0
	}

	// Get the node ID from the virtual node
	nodeID := h.virtualNodes[h.sortedHashes[idx]]
	node, exists := h.nodes[nodeID]

	if !exists {
		// This should never happen if internal state is consistent
		return nil, fmt.Errorf("internal error: virtual node points to non-existent node %s", nodeID)
	}

	// If the node is not active, find the next active node
	if !node.IsAvailable() {
		return h.getNextAvailableNode(idx)
	}

	return node, nil
}

// getNextAvailableNode finds the next available node starting from the given index
func (h *HashRing) getNextAvailableNode(startIdx int) (*Node, error) {
	// Create a copy of the sorted hashes to avoid issues with concurrent modification
	hashes := make([]uint32, len(h.sortedHashes))
	copy(hashes, h.sortedHashes)

	// Try each node in the ring
	for i := 0; i < len(hashes); i++ {
		idx := (startIdx + i) % len(hashes)
		nodeID := h.virtualNodes[hashes[idx]]

		// Skip virtual nodes pointing to the same physical node we already checked
		if i > 0 && nodeID == h.virtualNodes[hashes[(startIdx+i-1)%len(hashes)]] {
			continue
		}

		node, exists := h.nodes[nodeID]
		if exists && node.IsAvailable() {
			return node, nil
		}
	}

	return nil, errors.New("no active nodes available")
}

// GetNodes returns all nodes in the hash ring
func (h *HashRing) GetNodes() []*Node {
	h.mu.RLock()
	defer h.mu.RUnlock()

	nodes := make([]*Node, 0, len(h.nodes))
	for _, node := range h.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNodeCount returns the number of nodes in the hash ring
func (h *HashRing) GetNodeCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.nodes)
}

// UpdateNodeStatus updates a node's status
func (h *HashRing) UpdateNodeStatus(nodeID string, status NodeStatus) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	node, exists := h.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	node.SetStatus(status)
	return nil
}
