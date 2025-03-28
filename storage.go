package pantheon

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Storage struct {
	prefix    string
	namespace string
	redis     RedisClient
}

func NewStorage(prefix string, namespace string, client RedisClient) *Storage {
	return &Storage{
		prefix:    prefix,
		namespace: namespace,
		redis:     client,
	}
}

// makeKey creates a key for the storage
func (s *Storage) makeKey(parts ...string) string {
	return fmt.Sprintf("%s:%s:%s", s.prefix, s.namespace, strings.Join(parts, ":"))
}

// AddNode adds a new node to the cluster
// The node is identified by its ID.
// The address and port are used to communicate with the node.
// The path is the path on the node to make the heartbeat request to.
// The node is added with the state "alive".
// The node is added with the current time as the joined_at and last_heartbeat times.
func (s *Storage) AddNode(ctx context.Context, nodeID, address, path string, port int) error {
	key := s.makeKey("nodes", nodeID)

	joinedAt := fmt.Sprintf("%d", time.Now().Unix())

	nodeAddress := fmt.Sprintf("%s:%d", address, port)

	reply := s.redis.HSet(ctx, key,
		"address", nodeAddress,
		"path", path,
		"joined_at", joinedAt,
		"last_heartbeat", joinedAt,
		"hearbeat_count", "0",
		"heartbeat_failure_count", "0",
		"state", MemberAlive)
	if err := reply.Err(); err != nil {
		return err
	}

	return nil
}

// UpdateNodeHeartbeat updates the last heartbeat time for a node
func (s *Storage) UpdateNodeHeartbeat(ctx context.Context, nodeID string) error {
	key := s.makeKey("nodes", nodeID)

	lastHeartbeat := fmt.Sprintf("%d", time.Now().Unix())

	reply := s.redis.HSet(ctx, key, "last_heartbeat", lastHeartbeat)
	if err := reply.Err(); err != nil {
		return err
	}

	return nil
}

// UpdateNodeState updates the state of a node
func (s *Storage) UpdateNodeState(ctx context.Context, nodeID, state string) error {
	key := s.makeKey("nodes", nodeID)

	reply := s.redis.HSet(ctx, key, "state", state)
	if err := reply.Err(); err != nil {
		return err
	}

	return nil
}

func (s *Storage) IncrementHeartbeats(ctx context.Context, nodeID string) error {
	key := s.makeKey("nodes", nodeID)

	reply := s.redis.HIncrBy(ctx, key, "hearbeat_count", 1)
	if err := reply.Err(); err != nil {
		return err
	}

	return nil
}

func (s *Storage) IncrementHeartbeatFailures(ctx context.Context, nodeID string) error {
	key := s.makeKey("nodes", nodeID)

	reply := s.redis.HIncrBy(ctx, key, "hearbeat_failure_count", 1)
	if err := reply.Err(); err != nil {
		return err
	}

	return nil
}

func (s Storage) ResetHeartbeatFailures(ctx context.Context, nodeID string) error {
	key := s.makeKey("nodes", nodeID)

	reply := s.redis.HSet(ctx, key, "heartbeat_failure_count", "0")
	if err := reply.Err(); err != nil {
		return err
	}

	return nil
}

// GetNode retrieves a node from the cluster
func (s *Storage) GetNode(ctx context.Context, nodeID string) (*Member, error) {
	key := s.makeKey("nodes", nodeID)

	reply := s.redis.HGetAll(ctx, key)
	if err := reply.Err(); err != nil {
		return nil, err
	}

	value := reply.Val()

	if len(value) == 0 {
		return nil, nil
	}

	address, ok := value["address"]
	if !ok {
		return nil, NewErrNodePropertyNotFound("address")
	}

	path, ok := value["path"]
	if !ok {
		return nil, NewErrNodePropertyNotFound("path")
	}

	joinedAt, ok := value["joined_at"]
	if !ok {
		return nil, NewErrNodePropertyNotFound("joined_at")
	}

	lastHeartbeat, ok := value["last_heartbeat"]
	if !ok {
		return nil, NewErrNodePropertyNotFound("last_heartbeat")
	}

	heartbeatCount, ok := value["hearbeat_count"]
	if !ok {
		return nil, NewErrNodePropertyNotFound("hearbeat_count")
	}

	heartbeatFailures, ok := value["heartbeat_failure_count"]
	if !ok {
		return nil, NewErrNodePropertyNotFound("heartbeat_failure_count")
	}

	state, ok := value["state"]
	if !ok {
		return nil, NewErrNodePropertyNotFound("state")
	}

	member := &Member{
		ID:                nodeID,
		Address:           address,
		Path:              path,
		JoinedAt:          joinedAt,
		LastHeartbeat:     lastHeartbeat,
		HeartbeatCount:    heartbeatCount,
		HeartbeatFailures: heartbeatFailures,
		State:             MemberState(state),
	}

	return member, nil
}

// GetNodes retrieves all nodes from the cluster
func (s *Storage) GetNodes(ctx context.Context) ([]Member, error) {
	pattern := s.makeKey("nodes", "*")
	keys := s.redis.Keys(ctx, pattern)
	if err := keys.Err(); err != nil {
		return nil, err
	}

	nodeKeys := keys.Val()

	members := make([]Member, 0, len(nodeKeys))

	for _, key := range nodeKeys {
		nodeId := strings.TrimPrefix(key, s.makeKey("nodes", ""))
		member, err := s.GetNode(ctx, nodeId)
		if err != nil {
			return nil, err
		}

		members = append(members, *member)
	}

	return members, nil
}

// RemoveNode removes a node from the cluster
func (s *Storage) RemoveNode(ctx context.Context, nodeID string) error {
	// Get node keys before deleting the node
	nodeKeysKey := s.makeKey("nodekeys", nodeID)
	keys, err := s.redis.SMembers(ctx, nodeKeysKey).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("error getting node keys: %w", err)
	}

	// Remove key mappings for this node
	for _, key := range keys {
		keyMapKey := s.makeKey("keymap", key)
		if err := s.redis.Del(ctx, keyMapKey).Err(); err != nil {
			return fmt.Errorf("error removing key mapping: %w", err)
		}
	}

	// Remove the node keys set
	if err := s.redis.Del(ctx, nodeKeysKey).Err(); err != nil {
		return fmt.Errorf("error removing node keys: %w", err)
	}

	// Remove the node
	key := s.makeKey("nodes", nodeID)
	reply := s.redis.Del(ctx, key)
	if err := reply.Err(); err != nil {
		return err
	}

	return nil
}
