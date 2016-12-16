package blackfish

import (
	"net"
)

// Yo dawg. I herd you like bitwise arithmetic.

const (
	VERB_PING    = byte(0x00)
	VERB_ACK     = byte(0x01)
	VERB_NFPING  = byte(0x02)
	VERB_FORWARD = byte(0x03)

	STATUS_FORWARD_TO = byte(0x00)
	STATUS_JOINED     = byte(0x01)
	STATUS_LEFT       = byte(0x02)
	STATUS_DIED       = byte(0x03)
	STATUS_SUSPECTED  = byte(0x04)
	STATUS_ALIVE      = byte(0x05)
)

type message struct {
	_sender     *Node
	_senderCode uint32
	_verb       byte
	_members    []*messageMember
}

func newMessage(verb byte, sender *Node, code uint32) message {
	return message{
		_sender:     sender,
		_senderCode: code,
		_verb:       verb,
	}
}

type messageMember struct {
	code   uint32
	node   *Node
	status byte
}

func (m *message) addMember(status byte, n *Node, code uint32) {
	if m._members == nil {
		m._members = make([]*messageMember, 0, 32)
	}

	messageMember := messageMember{
		code:   code,
		node:   n,
		status: status}

	m._members = append(m._members, &messageMember)
}

func (m *message) setSender(n *Node) {
	m._sender = n

}

func (m *message) getSender() *Node {
	return m._sender
}

func (m *message) setSenderCode(n uint32) {
	m._senderCode = n

}

func (m *message) getSenderCode() uint32 {
	return m._senderCode
}

func (m *message) setVerb(v byte) {
	m._verb = v

}

func (m *message) getVerb() byte {
	return m._verb
}

func (m *message) getForwardTo() *messageMember {
	if len(m._members) > 0 && m._members[0].status == STATUS_FORWARD_TO {
		return m._members[0]
	} else {
		return nil
	}
}

// Message contents (byte, content)
// Bytes Content
// Bytes 00    Verb (one of {P|A|F|N})
// Bytes 01-02 Sender response port
// Bytes 03-06 Sender ID Code
// Bytes 07    Member status byte
// Bytes 08-11 Member host IP
// Bytes 12-13 Member host response port
// Bytes 14-17 Member message code

func (m *message) encode() []byte {
	bytes := make([]byte, 17, 17)

	// Bytes 00    Verb (one of {P|A|F|N})
	// Translation: the first character of the message verb
	bytes[0] = m._verb

	// Bytes 01-02 Sender response port
	sport := m._sender.Port
	for i := uint16(0); i < 2; i++ {
		sport >>= (i * 8)
		bytes[i+1] = byte(sport)
	}

	// Bytes 03-06 ID Code
	scode := m._senderCode
	for i := uint32(0); i < 4; i++ {
		scode >>= (i * 8)
		bytes[i+3] = byte(scode)
	}

	// byte pointer
	p := 7

	// Each member data requires 10 bytes.
	for _, member := range m._members {
		mnode := member.node
		mstatus := member.status
		mcode := member.code

		// Byte p + 0
		bytes[p] = mstatus

		// Bytes (p + 01) to (p + 04): Originating host IP
		ipb := mnode.IP
		for i := 0; i < 4; i++ {
			bytes[p+1+i] = ipb[i]
		}

		// Bytes (p + 05) to (p + 06): Originating host response port
		oport := mnode.Port
		for i := 0; i < 2; i++ {
			oport >>= uint16(i * 8)
			bytes[p+5+i] = byte(oport)
		}

		// Bytes (p + 07) to (p + 10): Originating message code
		ocode := mcode
		for i := 0; i < 4; i++ {
			ocode >>= uint32(i * 8)
			bytes[p+7+i] = byte(ocode)
		}

		p += 11
	}

	return bytes
}

// Parses the bytes received in a UDP message.
// If the address:port from the message can't be associated with a known
// (live) node, then the value of message.sender will be nil.
func decodeMessage(addr *net.UDPAddr, bytes []byte) (message, error) {
	// Message contents (byte, content)
	// Bytes Content
	// Bytes 00    Verb (one of {P|A|F|N})
	// Bytes 01-02 Sender response port
	// Bytes 03-06 Sender ID Code
	// Bytes 07    Member status byte
	// Bytes 08-11 Member host IP
	// Bytes 12-13 Member host response port
	// Bytes 14-17 Member message code

	// Bytes 00    Verb (one of {P|A|F|N})
	var message_verb byte = bytes[0] & byte(0x03)

	// Bytes 01-02 Sender response port
	var sender_port uint16
	for i := 2; i >= 1; i-- {
		sender_port <<= 8
		sender_port |= uint16(bytes[i])
	}

	// Bytes 03-06 Sender ID Code
	var sender_code uint32
	for i := 6; i >= 3; i-- {
		sender_code <<= 8
		sender_code |= uint32(bytes[i])
	}

	// Now that we have the IP and port, we can find the Node.
	sender := live_nodes.GetByIP(addr.IP.To4(), sender_port)

	// We don't know this node, so create a new one!
	if sender == nil {
		_, sender, _ = live_nodes.AddByIP(addr.IP.To4(), sender_port)
	}

	// Now that we have the verb, node, and code, we can build the mesage
	m := newMessage(message_verb, sender, sender_code)

	// Byte 00 also contains the number of piggybacked messages
	// in the leftmost 6 bits
	// member_update_count := int(bytes[0] >> 2)

	m._members = parseMembers(bytes[7:])

	return m, nil
}

func parseMembers(bytes []byte) []*messageMember {
	members := make([]*messageMember, 0, 1)

	for b := 0; b < len(bytes); b += 10 {
		var mstatus byte
		var mip net.IP
		var mport uint16
		var mcode uint32
		var mnode *Node

		// Byte 00 Member status byte
		mstatus = bytes[0]

		// Bytes 01-04 Originating host IP (FWD only)
		if bytes[b+0] > 0 {
			mip = net.IPv4(
				bytes[b+0],
				bytes[b+1],
				bytes[b+2],
				bytes[b+3]).To4()
		}

		// Bytes 05-06 Originating host response port (FWD only)
		for i := 5; i >= 4; i-- {
			mport <<= 8
			mport |= uint16(bytes[b+i])
		}

		// Bytes 07-10 Originating message code (FWD only)
		for i := 9; i >= 6; i-- {
			mcode <<= 8
			mcode |= uint32(bytes[b+i])
		}

		if len(mip) > 0 {
			// Find the sender by the address associated with the message actual
			mnode = live_nodes.GetByIP(mip, mport)

			// We don't know this node, so create a new one!
			if mnode == nil {
				_, mnode, _ = live_nodes.AddByIP(mip, mport)
			}
		}

		member := messageMember{
			code:   mcode,
			node:   mnode,
			status: mstatus,
		}

		members = append(members, &member)
	}

	return members
}
