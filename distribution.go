package pantheon

import "fmt"

// Distribute distributes keys to the nodes in the cluster
// This should be called after a nodes has joined or left the cluster to
// rebalance the keys to available nodes
func (c *Pantheon) Distribute(keys []string) error {
	if !c.started {
		return fmt.Errorf("cluster not started")
	}

	// Check if hashring is available
	if c.hashRing == nil {
		return fmt.Errorf("hash ring not initialized")
	}

	// Check if there are nodes in the hash ring
	if c.hashRing.GetNodeCount() == 0 {
		return fmt.Errorf("no nodes in the hash ring")
	}

	fmt.Printf("Distributing %d keys using consistent hashing\n", len(keys))

	// Use consistent hashing to distribute keys
	distribution := make(map[string][]string)
	
	for _, key := range keys {
		// Get the node for this key using consistent hashing
		node, err := c.hashRing.GetNode(key)
		if err != nil {
			return fmt.Errorf("error getting node for key %s: %w", key, err)
		}
		
		// Initialize the slice if it doesn't exist
		if distribution[node.ID] == nil {
			distribution[node.ID] = make([]string, 0)
		}
		
		// Add the key to the node's distribution
		distribution[node.ID] = append(distribution[node.ID], key)

		// Store the key-to-node mapping in Redis
		keyMapKey := c.storage.makeKey("keymap", key)
		if err := c.storage.redis.Set(c.ctx, keyMapKey, node.ID, 0).Err(); err != nil {
			return fmt.Errorf("error storing key mapping: %w", err)
		}

		// Also store the key in a set for each node for faster retrieval
		nodeKeysKey := c.storage.makeKey("nodekeys", node.ID)
		if err := c.storage.redis.SAdd(c.ctx, nodeKeysKey, key).Err(); err != nil {
			return fmt.Errorf("error storing node key: %w", err)
		}
	}

	// Log the distribution
	fmt.Printf("Distribution results:\n")
	for nodeID, assignedKeys := range distribution {
		fmt.Printf("Node %s assigned %d keys\n", nodeID, len(assignedKeys))
	}

	return nil
}

// GetNodeKeys retrieves the keys assigned to a node.
// This is used to determine which keys a node in the cluster is responsible for
func (c *Pantheon) GetNodeKeys(nodeID string) ([]string, error) {
	if !c.started {
		return nil, fmt.Errorf("cluster not started")
	}

	// Check if the node exists in the hash ring
	exists := false
	nodes := c.hashRing.GetNodes()
	for _, node := range nodes {
		if node.ID == nodeID {
			exists = true
			break
		}
	}

	if !exists {
		return nil, fmt.Errorf("node %s not found in hash ring", nodeID)
	}

	// Get the keys from Redis
	nodeKeysKey := c.storage.makeKey("nodekeys", nodeID)
	result, err := c.storage.redis.SMembers(c.ctx, nodeKeysKey).Result()
	if err != nil {
		if err == redis.Nil {
			// No keys found, return empty slice
			return []string{}, nil
		}
		return nil, fmt.Errorf("error getting keys for node %s: %w", nodeID, err)
	}

	return result, nil
}

// GetKeyNode returns the node responsible for a specific key
func (c *Pantheon) GetKeyNode(key string) (string, error) {
	if !c.started {
		return "", fmt.Errorf("cluster not started")
	}

	// First check if the key is already mapped in Redis
	keyMapKey := c.storage.makeKey("keymap", key)
	nodeID, err := c.storage.redis.Get(c.ctx, keyMapKey).Result()
	if err != nil && err != redis.Nil {
		return "", fmt.Errorf("error getting node for key %s: %w", key, err)
	}

	// If the key is already mapped, return the node ID
	if err == nil && nodeID != "" {
		return nodeID, nil
	}

	// Otherwise, use the hash ring to determine the node
	node, err := c.hashRing.GetNode(key)
	if err != nil {
		return "", fmt.Errorf("error determining node for key %s: %w", key, err)
	}

	// Store the mapping for future use
	if err := c.storage.redis.Set(c.ctx, keyMapKey, node.ID, 0).Err(); err != nil {
		return "", fmt.Errorf("error storing key mapping: %w", err)
	}

	// Also add to the node's key set
	nodeKeysKey := c.storage.makeKey("nodekeys", node.ID)
	if err := c.storage.redis.SAdd(c.ctx, nodeKeysKey, key).Err(); err != nil {
		return "", fmt.Errorf("error storing node key: %w", err)
	}

	return node.ID, nil
}
