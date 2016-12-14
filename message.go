package blackfish

import (
	"errors"
	"net"
)

type message struct {
	verb           string
	sender         *Node
	senderIP       net.IP
	senderPort     uint16
	senderCode     uint32
	downstream     *Node
	downstreamIP   net.IP
	downstreamPort uint16
	downstreamCode uint32
}

// Message contents (byte, content)
// Bytes Content
// Bytes 00    Verb (one of {P|A|F|N})
// Bytes 01-02 Sender response port
// Bytes 03-06 Sender ID Code
// Bytes 07-10 Originating host IP (FWD only)
// Bytes 11-12 Originating host response port (FWD only)
// Bytes 13-16 Originating message code (FWD only)

func encodeMessage(msg message) []byte {
	bytes := make([]byte, 17, 17)

	// Bytes 00    Verb (one of {P|A|F|N})
	// Translation: the first character of the message verb
	bytes[0] = []byte(msg.verb)[0]

	// Bytes 01-02 Sender response port
	sport := msg.senderPort
	for i := uint16(0); i < 2; i++ {
		sport >>= (i * 8)
		bytes[i+1] = byte(sport)
	}

	// Bytes 03-06 ID Code
	scode := msg.senderCode
	for i := uint32(0); i < 4; i++ {
		scode >>= (i * 8)
		bytes[i+3] = byte(scode)
	}

	// Bytes 07-10 Originating host IP (FWD only)
	if msg.downstreamIP != nil {
		ipb := []byte(msg.downstreamIP)

		for i := 0; i < 4; i++ {
			bytes[i+7] = ipb[i]
		}
	}

	// Bytes 11-12 Originating host response port (FWD only)
	oport := msg.downstreamPort
	for i := uint16(0); i < 2; i++ {
		oport >>= (i * 8)
		bytes[i+11] = byte(oport)
	}

	// Bytes 13-16 Originating message code (FWD only)
	ocode := msg.downstreamCode
	for i := uint32(0); i < 2; i++ {
		ocode >>= (i * 8)
		bytes[i+13] = byte(ocode)
	}

	return bytes
}

// Parses the bytes received in a UDP message.
// If the address:port from the message can't be associated with a known
// (live) node, then the value of message.sender will be nil.
func decodeMessage(addr *net.UDPAddr, bytes []byte) (message, error) {
	msg := message{}

	// Message contents (byte, content)
	// Bytes Content
	// Bytes 00    Verb (one of {P|A|F|N})
	// Bytes 01-02 Sender response port
	// Bytes 03-06 Sender ID Code
	// Bytes 07-10 Originating host IP (FWD only)
	// Bytes 11-12 Originating host response port (FWD only)
	// Bytes 13-16 Originating message code (FWD only)

	// Bytes 00    Verb (one of {P|A|F|N})
	verb_char := string(bytes[0:1])
	switch {
	case verb_char == "P":
		msg.verb = "PING"
	case verb_char == "A":
		msg.verb = "ACK"
	case verb_char == "F":
		msg.verb = "FORWARD"
	case verb_char == "N":
		msg.verb = "NFPING"
	}

	if msg.verb == "" {
		return msg, errors.New("Verb not found")
	}

	msg.senderIP = addr.IP.To4()

	// Bytes 01-02 Sender response port
	for i := 2; i >= 1; i-- {
		msg.senderPort <<= 8
		msg.senderPort |= uint16(bytes[i])
	}

	// Bytes 03-06 Sender ID Code
	for i := 6; i >= 3; i-- {
		msg.senderCode <<= 8
		msg.senderCode |= uint32(bytes[i])
	}

	// Find the sender by the address associated with the message actual
	msg.sender = GetNodeByIP(msg.senderIP, msg.senderPort)

	// Bytes 07-10 Originating host IP (FWD only)
	if bytes[7] > 0 {
		msg.downstreamIP = net.IPv4(bytes[7], bytes[8], bytes[9], bytes[10]).To4()
	}

	// Bytes 11-12 Originating host response port (FWD only)
	for i := 12; i >= 11; i-- {
		msg.downstreamPort <<= 8
		msg.downstreamPort |= uint16(bytes[i])
	}

	// Bytes 13-16 Originating message code (FWD only)
	for i := 16; i >= 13; i-- {
		msg.downstreamCode <<= 8
		msg.downstreamCode |= uint32(bytes[i])
	}

	if len(msg.downstreamIP) > 0 {
		// Find the sender by the address associated with the message actual
		msg.downstream = GetNodeByIP(msg.downstreamIP, msg.downstreamPort)

		if msg.downstream == nil {
			msg.downstream = AddNodeByIP(msg.downstreamIP, msg.downstreamPort)
		}
	}

	return msg, nil
}
