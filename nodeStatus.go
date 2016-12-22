package blackfish

// NodeStatus represents the believed status of a member node.
type NodeStatus byte

const (
	// StatusUnknown is the default node status of newly-created nodes.
	StatusUnknown NodeStatus = iota

	// StatusAlive indicates tthat a node is alive and healthy.
	StatusAlive

	// StatusDead indicatates that a node is dead and no longer healthy.
	StatusDead

	// StatusForwardTo is a pseudo status used by message to indicate
	// the target of a ping request.
	StatusForwardTo
)

func (s NodeStatus) String() string {
	switch s {
	case StatusUnknown:
		return "UNKNOWN"
	case StatusAlive:
		return "ALIVE"
	case StatusDead:
		return "DEAD"
	case StatusForwardTo:
		return "FORWARD_TO"
	default:
		return "UNDEFINED"
	}
}
