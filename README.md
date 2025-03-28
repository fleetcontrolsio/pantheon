# Pantheon

Pantheon is a toolkit for distributing workloads across distributed nodes using a consistent hashring.

## Features

- Consistent hashing algorithm for even workload distribution
- Fault-tolerant node management with health monitoring
- Automatic node failover when nodes become unavailable
- HTTP-based heartbeat mechanism with Redis-based persistence
- Event-driven architecture with callbacks for node status changes

## Requirements

- Go 1.24 or higher
- Redis server

## Installation

```bash
go get github.com/fleetcontrolsio/pantheon
```

## Usage

### Creating a Pantheon Cluster

```go
// Create a context
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Create a custom HTTP client with timeout
httpClient := &http.Client{
    Timeout: 5 * time.Second,
}

// Create options
options := pantheon.NewOptions().
    WithName("my-cluster").
    WithHeartbeatInterval(10 * time.Second).
    WithHeartbeatTimeout(5 * time.Second).
    WithHTTPClient(httpClient)

// Create Pantheon instance
p, err := pantheon.New(ctx, options)
if err != nil {
    // Handle error
}

// Start pantheon
if err := p.Start(); err != nil {
    // Handle error
}
```

### Adding Nodes to the Cluster

```go
// Add a node to the cluster
err = p.Join(&pantheon.JoinOp{
    ID:      "node-1",
    Address: "http://localhost",
    Port:    8080,
    Path:    "health",
})
if err != nil {
    // Handle error
}
```

### Node Health Monitoring

Pantheon automatically monitors the health of nodes by sending HTTP requests to the specified health endpoint. When a node fails to respond, it is marked as suspect. After multiple failures, it is marked as dead and removed from the available nodes list.

```go
// Listen for node events
go func() {
    for event := range p.EventsCh {
        switch event.Event {
        case "died":
            fmt.Printf("Node %s is considered dead\n", event.NodeID)
        case "revived":
            fmt.Printf("Node %s has recovered\n", event.NodeID)
        }
    }
}()
```

### Distributing Keys with Consistent Hashing

```go
// Distribute some keys to available nodes using consistent hashing
keys := []string{"user:1", "user:2", "user:3", "product:1", "product:2"}
err = p.Distribute(keys)
if err != nil {
    // Handle error
}

// Get keys assigned to a specific node
nodeKeys, err := p.GetNodeKeys("node-1")
if err != nil {
    // Handle error
}
fmt.Printf("Node 1 is responsible for %d keys: %v\n", len(nodeKeys), nodeKeys)

// Find which node is responsible for a specific key
nodeID, err := p.GetKeyNode("user:1")
if err != nil {
    // Handle error
}
fmt.Printf("Key 'user:1' is assigned to node: %s\n", nodeID)
```

When nodes join or leave the cluster, keys are automatically redistributed using the consistent hashing algorithm, minimizing the number of keys that need to be remapped.

### Graceful Shutdown

```go
// Remove a node from the cluster (graceful shutdown)
err = p.Leave("node-1")
if err != nil {
    // Handle error
}

// Destroy the Pantheon instance
if err := p.Destroy(); err != nil {
    // Handle error
}
```

## Architecture

Pantheon uses Redis for persistence, HTTP for communication, and consistent hashing for workload distribution:

1. **Node Management**: 
   - Nodes register with the Pantheon cluster with a unique ID and health endpoint
   - Pantheon periodically sends HTTP requests to each node's health endpoint
   - Nodes that fail to respond are marked as suspect and eventually dead
   - When node status changes, events are sent through the EventsCh channel

2. **Consistent Hashing**:
   - Keys are distributed across available nodes using a consistent hash ring
   - Each physical node gets multiple virtual nodes on the ring for better distribution
   - When nodes are added/removed, only a minimal fraction of keys need to be remapped
   - The hash ring automatically routes around failed nodes

3. **Persistence**:
   - Redis stores node information, heartbeat data, and key-to-node mappings
   - Key mappings are stored both as direct lookups and as sets per node
   - Node state (alive, suspect, dead) is tracked for proper failover handling

## License

see [license](LICENSE)