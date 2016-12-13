package blackfish

import (
	"errors"
	"net"
)

type message struct {
	verb   string
	code   uint32
	sender *Node
	origin *Node
}

func encodeMessage(msg message) []byte {
	bytes_out := []byte(msg.verb)

	bytes_out = append(bytes_out, byte(0))

	code := msg.code
	for i := uint32(0); i < 32; i += 8 {
		code >>= i
		bytes_out = append(bytes_out, byte(code))
	}

	return bytes_out
}

// Parses the bytes received in a UDP message.
// If the address:port from the message can't be associated with a known
// (live) node, then the value of message.sender will be nil.
func decodeMessage(addr *net.UDPAddr, msg_bytes []byte) (message, error) {
	msg := message{}

	// TODO
	// Message contents (byte, content)
	// Bytes Content
	// 00-02 Verb
	// 03-06 ID Code
	// 07-08 Sender response port
	// 09-12 Originating host IP (FWD only)
	// 13-14 Originating host response port (FWD only)
	// 15-18 Originating message code (FWD only)

	// Scan to find the null byte and use that to find the verb
	for i, b := range msg_bytes {
		if b == 0 {
			msg.verb = string(msg_bytes[:i])
			break
		}
	}

	// Everything after is the code value
	for j := len(msg_bytes) - 1; j > len(msg.verb); j-- {
		msg.code <<= 8
		msg.code |= uint32(msg_bytes[j])
	}

	if msg.verb == "" {
		return msg, errors.New("Verb not found")
	}

	// Find the sender by the address associated with the message actual
	msg.sender = GetNodeByIP(addr.IP, 0) // TODO: Parse out port!

	return msg, nil
}
