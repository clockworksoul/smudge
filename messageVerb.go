package blackfish

type messageVerb byte

const (
	// VerbPing represents a simple ping. If this ping is not responded to with
	// an ack within a timeout period, the pinging host will attempt to ping
	// indirectly via one or more additional hosts with a ping request.
	VerbPing messageVerb = iota

	// VerbAck represents a response to a ping request.
	VerbAck

	// VerbPingRequest represents a request made by one host to another to ping
	// a third host whose live status is in question.
	VerbPingRequest

	// VerbNonForwardingPing represents a ping in response to a ping request.
	// If the ping times out, the host does not follow up with a ping request
	// to any other hosts.
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
