package pantheon

type MemberState string

const (
	MemberAlive   MemberState = "alive"
	MemberDead    MemberState = "dead"
	MemberSuspect MemberState = "suspect"
)

// MarshalBinary implements the encoding.BinaryMarshaler interface
func (m MemberState) MarshalBinary() (data []byte, err error) {
	return []byte(m), nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface
func (m *MemberState) UnmarshalBinary(data []byte) error {
	*m = MemberState(data)
	return nil
}

// Member represents a node in the cluster
type Member struct {
	// ID; the unique identifier for the node
	ID string
	// Address; the address of the node
	Address string
	// Path; the path on the node to make the heartbeat request to
	Path string
	// JoinedAt; the time the node joined the cluster
	JoinedAt string
	// LastHeartbeat; the last time a heartbeat was received from the node
	LastHeartbeat string
	// HearbeatCount; the number of heartbeat requests sent to the node
	HeartbeatCount string
	// HeartbeatFailures; the number of failed heartbeat requests
	HeartbeatFailures string
	// State; the state of the node: alive, dead, or suspect
	State MemberState
}
