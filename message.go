/*
Copyright 2016 The Smudge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package smudge

import (
	"errors"
	"net"
)

type message struct {
	sender     *Node
	senderCode uint32
	verb       messageVerb
	members    []*messageMember
}

// Represents a "member" of a message; i.e., a node that the sender knows
// about, about which it wishes to notify the downstream recipient.
type messageMember struct {
	code   uint32
	node   *Node
	status NodeStatus
}

func (m *message) addMember(n *Node, status NodeStatus, code uint32) {
	if m.members == nil {
		m.members = make([]*messageMember, 0, 32)
	}

	messageMember := messageMember{
		code:   code,
		node:   n,
		status: status}

	m.members = append(m.members, &messageMember)
}

// Message contents (byte, content)
// Bytes Content
// Bytes 00    Verb (one of {P|A|F|N})
// Bytes 01-02 Sender response port
// Bytes 03-06 Sender ID Code
// ---
// Bytes 07    Member status byte
// Bytes 08-11 Member host IP
// Bytes 12-13 Member host response port
// Bytes 14-17 Member message code

func (m *message) encode() []byte {
	size := 7 + (len(m.members) * 11)
	bytes := make([]byte, size, size)

	// Bytes 00    Verb (one of {P|A|F|N})
	// Translation: the first character of the message verb
	bytes[0] = byte(m.verb)

	// Bytes 01-02 Sender response port
	sport := m.sender.port
	for i := uint16(0); i < 2; i++ {
		sport >>= (i * 8)
		bytes[i+1] = byte(sport)
	}

	// Bytes 03-06 ID Code
	scode := m.senderCode
	for i := uint32(0); i < 4; i++ {
		scode >>= (i * 8)
		bytes[i+3] = byte(scode)
	}

	// byte pointer
	p := 7

	// Each member data requires 11 bytes.
	for _, member := range m.members {
		mnode := member.node
		mstatus := member.status
		mcode := member.code

		// Byte p + 00
		bytes[p] = byte(mstatus)

		// Bytes (p + 01) to (p + 04): Originating host IP
		ipb := mnode.ip
		for i := 0; i < 4; i++ {
			bytes[p+1+i] = ipb[i]
		}

		// Bytes (p + 05) to (p + 06): Originating host response port
		oport := mnode.port
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

	if (len(bytes)-7)%11 != 0 {
		logfWarn("Inconsistent byte count received from %v: %d MOD 11 = %d\n",
			addr.IP,
			len(bytes),
			len(bytes)%11)

		return newMessage(255, nil, 0),
			errors.New("unexpected byte length received in message")
	}

	// Bytes 00    Verb (one of {P|A|F|N})
	verb := messageVerb(bytes[0] & byte(0x03))

	// Bytes 01-02 Sender response port
	var senderPort uint16
	for i := 2; i >= 1; i-- {
		senderPort <<= 8
		senderPort |= uint16(bytes[i])
	}

	// Bytes 03-06 Sender ID Code
	var senderCode uint32
	for i := 6; i >= 3; i-- {
		senderCode <<= 8
		senderCode |= uint32(bytes[i])
	}

	// Now that we have the IP and port, we can find the Node.
	sender := knownNodes.getByIP(addr.IP.To4(), senderPort)

	// We don't know this node, so create a new one!
	if sender == nil {
		sender, _ = CreateNodeByIP(addr.IP.To4(), senderPort)
	}

	// Now that we have the verb, node, and code, we can build the mesage
	m := newMessage(verb, sender, senderCode)

	// Byte 00 also contains the number of piggybacked messages
	// in the leftmost 6 bits
	// member_update_count := int(bytes[0] >> 2)

	if len(bytes) > 7 {
		m.members = parseMembers(bytes[7:])
	}

	return m, nil
}

// If members exist on this message, and that message has the "forward to"
// status, this function returns it; otherwise it returns nil.
func (m *message) getForwardTo() *messageMember {
	if len(m.members) > 0 && m.members[0].status == StatusForwardTo {
		return m.members[0]
	}

	return nil
}

// Convenience function. Creates a new message instance.
func newMessage(verb messageVerb, sender *Node, code uint32) message {
	return message{
		sender:     sender,
		senderCode: code,
		verb:       verb,
	}
}

func parseMembers(bytes []byte) []*messageMember {
	// Bytes 00    Member status byte
	// Bytes 01-04 Member host IP
	// Bytes 05-06 Member host response port
	// Bytes 07-10 Member message code

	members := make([]*messageMember, 0, 1)

	for b := 0; b < len(bytes); b += 11 {
		var mstatus NodeStatus
		var mip net.IP
		var mport uint16
		var mcode uint32
		var mnode *Node

		// Byte 00 Member status byte
		mstatus = NodeStatus(bytes[b+0])

		// Bytes 01-04 member IP
		if bytes[b+1] > 0 {
			mip = net.IPv4(
				bytes[b+1],
				bytes[b+2],
				bytes[b+3],
				bytes[b+4]).To4()
		}

		// Bytes 05-06 member response port
		for i := 6; i >= 5; i-- {
			mport <<= 8
			mport |= uint16(bytes[b+i])
		}

		// Bytes 07-10 member message code
		for i := 10; i >= 7; i-- {
			mcode <<= 8
			mcode |= uint32(bytes[b+i])
		}

		if len(mip) > 0 {
			// Find the sender by the address associated with the message
			mnode = knownNodes.getByIP(mip, mport)

			// We still don't know this node, so create a new one!
			if mnode == nil {
				mnode, _ = CreateNodeByIP(mip, mport)
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
