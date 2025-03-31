package pantheon

type PantheonEvent struct {
	// Event; the name of the event
	// "started" - when the cluster is started
	// "joined" - when a node joins the cluster
	// "left" - when a node leaves the cluster
	// "died" - when a node is considered dead (no heartbeat received/timeout)
	Event string
	// NodeID; the identifier of the node
	NodeID string
}
