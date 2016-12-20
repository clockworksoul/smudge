package blackfish

type messageVerb byte

const (
	VERB_PING messageVerb = iota
	VERB_ACK
	VERB_NFPING
	VERB_FORWARD
)

func (v messageVerb) String() string {
	switch v {
	case VERB_PING:
		return "PING"
	case VERB_ACK:
		return "ACK"
	case VERB_FORWARD:
		return "FORWARD"
	case VERB_NFPING:
		return "NFPING"
	default:
		return "UNDEFINED"
	}
}
