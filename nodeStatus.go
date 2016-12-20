package blackfish

type NodeStatus byte

const (
	STATUS_NONE NodeStatus = iota
	STATUS_ALIVE
	STATUS_SUSPECTED
	STATUS_LEFT
	STATUS_DIED
	STATUS_FORWARD_TO
)

func (s NodeStatus) String() string {
	switch s {
	case STATUS_NONE:
		return "NONE"
	case STATUS_ALIVE:
		return "ALIVE"
	case STATUS_FORWARD_TO:
		return "FORWARD_TO"
	case STATUS_LEFT:
		return "LEFT"
	case STATUS_DIED:
		return "DIED"
	case STATUS_SUSPECTED: // NOT YET IMPLEMENTED
		return "SUSPECTED"
	default:
		return "UNDEFINED"
	}
}
