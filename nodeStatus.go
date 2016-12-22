package blackfish

type NodeStatus byte

const (
	StatusNone NodeStatus = iota
	StatusAlive
	StatusDead
	StatusForwardTo
)

func (s NodeStatus) String() string {
	switch s {
	case StatusNone:
		return "NONE"
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
