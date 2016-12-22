package blackfish

type messageVerb byte

const (
	VerbPing messageVerb = iota
	VerbAck
	VerbPingRequest
	VerbNonForwardingPing
)

func (v messageVerb) String() string {
	switch v {
	case VerbPing:
		return "PING"
	case VerbAck:
		return "ACK"
	case VerbPingRequest:
		return "PINGREQ"
	case VerbNonForwardingPing:
		return "NFPING"
	default:
		return "UNDEFINED"
	}
}
