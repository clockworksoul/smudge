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
	"hash/adler32"
	"net"
)

// Message contents
// ---[ Base message (11 bytes)]---
// Bytes 00-03 Checksum (32-bit)
// Bytes 04    Verb (one of {PING|ACK|PINGREQ|NFPING})
// Bytes 05-06 Sender response port
// Bytes 07-10 Sender ID Code
// ---[ Per member (11 bytes)]---
// Bytes 00    Member status byte
// Bytes 01-04 Member host IP
// Bytes 05-06 Member host response port
// Bytes 07-10 Member message code
// ---[ Per broadcast (1 allowed) (11+N bytes) ]
// Bytes 00-03 Origin IP
// Bytes 04-05 Origin response port
// Bytes 06-09 Origin broadcast counter
// Bytes 10-11 Payload length (bytes)
// Bytes 12-NN Payload

type message struct {
	sender     *Node
	senderCode uint32
	verb       messageVerb
	members    []*messageMember
	broadcast  *Broadcast
}

// Represents a "member" of a message; i.e., a node that the sender knows
// about, about which it wishes to notify the downstream recipient.
type messageMember struct {
	code   uint32
	node   *Node
	status NodeStatus
}

// Convenience function. Creates a new message instance.
func newMessage(verb messageVerb, sender *Node, code uint32) message {
	return message{
		sender:     sender,
		senderCode: code,
		verb:       verb,
	}
}

// Adds a broadcast to this message. Only one broadcast is allowed; subsequent
// calls will replace an existing broadcast.
func (m *message) addBroadcast(broadcast *Broadcast) {
	m.broadcast = broadcast
}

// Adds a member status update to this message. The maximum number of allowed
// members is 2^6 - 1 = 63, though it is incredibly unlikely that this maximum
// will be reached without an absurdly high lambda. There aren't yet many
// 88 billion node clusters (assuming lambda of 2.5).
func (m *message) addMember(n *Node, status NodeStatus, code uint32) error {
	if m.members == nil {
		m.members = make([]*messageMember, 0, 32)
	} else if len(m.members) >= 63 {
		return errors.New("member list overflow")
	}

	messageMember := messageMember{
		code:   code,
		node:   n,
		status: status}

	m.members = append(m.members, &messageMember)

	return nil
}

// Message contents
// ---[ Base message (12 bytes)]---
// Bytes 00-03 Checksum (32-bit)
// Bytes 04    Verb (one of {PING|ACK|PINGREQ|NFPING})
// Bytes 05-06 Sender response port
// Bytes 07-10 Sender ID Code
// ---[ Per member (11 bytes)]---
// Bytes 00    Member status byte
// Bytes 01-04 Member host IP
// Bytes 05-06 Member host response port
// Bytes 07-10 Member message code

func (m *message) encode() []byte {
	size := 11 + (len(m.members) * 11)
	if m.broadcast != nil {
		size += 12 + len(m.broadcast.bytes)
	}

	bytes := make([]byte, size, size)

	// An index pointer (start at 4 to accommodate checksum)
	p := 4

	// Byte 00
	// Rightmost 2 bits: verb (one of {P|A|F|N})
	// Leftmost 6 bits: number
	verbByte := byte(len(m.members))
	verbByte = (verbByte << 2) | byte(m.verb)
	p += encodeByte(verbByte, bytes, p)

	// Bytes 01-02 Sender response port
	p += encodeUint16(m.sender.port, bytes, p)

	// Bytes 03-06 ID Code
	p += encodeUint32(m.senderCode, bytes, p)

	// Each member data requires 11 bytes.
	for _, member := range m.members {
		mnode := member.node
		mstatus := member.status
		mcode := member.code

		// Byte p + 00
		bytes[p] = byte(mstatus)
		p++

		// Bytes (p + 01) to (p + 04): Originating host IP
		ipb := mnode.ip
		for i := 0; i < 4; i++ {
			bytes[p+i] = ipb[i]
		}
		p += 4

		// Bytes (p + 05) to (p + 06): Originating host response port
		p += encodeUint16(mnode.port, bytes, p)

		// Bytes (p + 07) to (p + 10): Originating message code
		p += encodeUint32(mcode, bytes, p)
	}

	if m.broadcast != nil {
		bbytes := m.broadcast.encode()
		for i, v := range bbytes {
			bytes[p+i] = v
		}
	}

	checksum := adler32.Checksum(bytes[4:])
	encodeUint32(checksum, bytes, 0)

	return bytes
}

// If members exist on this message, and that message has the "forward to"
// status, this function returns it; otherwise it returns nil.
func (m *message) getForwardTo() *messageMember {
	if len(m.members) > 0 && m.members[0].status == StatusForwardTo {
		return m.members[0]
	}

	return nil
}

// Parses the bytes received in a UDP message.
// If the address:port from the message can't be associated with a known
// (live) node, then an instance of message.sender will be created from
// available data but not explicitly added to the known nodes.
func decodeMessage(sourceIP net.IP, bytes []byte) (message, error) {
	var err error

	// An index pointer
	p := 0

	// Bytes 00-03 Checksum (32-bit)
	checksumStated, p := decodeUint32(bytes, p)
	checksumCalculated := adler32.Checksum(bytes[4:])
	if checksumCalculated != checksumStated {
		return newMessage(255, nil, 0),
			errors.New("checksum failure from " + sourceIP.String())
	}

	// Bytes 04    Verb (one of {P|A|F|N})
	v, p := decodeByte(bytes, p)
	verb := messageVerb(v & 0x03)

	memberCount := int(v >> 2)

	// Bytes 05-06 Sender response port
	senderPort, p := decodeUint16(bytes, p)

	// Bytes 07-10 Sender ID Code
	senderCode, p := decodeUint32(bytes, p)

	// Now that we have the IP and port, we can find the Node.
	sender := knownNodes.getByIP(sourceIP.To4(), senderPort)

	// We don't know this node, so create a new one!
	if sender == nil {
		sender, _ = CreateNodeByIP(sourceIP.To4(), senderPort)
	}

	// Now that we have the verb, node, and code, we can build the mesage
	m := newMessage(verb, sender, senderCode)

	memberLastIndex := p + (memberCount * 11)
	if len(bytes) > p {
		m.members = decodeMembers(memberCount, bytes[p:memberLastIndex])
	}

	if len(bytes) > memberLastIndex {
		m.broadcast, err = decodeBroadcast(bytes[memberLastIndex:])

		if m.broadcast.origin.IP()[0] == 0 || m.broadcast.origin.Port() == 0 {
			err = errors.New("Received originless broadcast!")
		}
	}

	return m, err
}

func decodeMembers(memberCount int, bytes []byte) []*messageMember {
	// Bytes 00    Member status byte
	// Bytes 01-04 Member host IP
	// Bytes 05-06 Member host response port
	// Bytes 07-10 Member message code

	members := make([]*messageMember, 0, 1)

	// An index pointer
	p := 0

	for p < len(bytes) {
		var mstatus NodeStatus
		var mip net.IP
		var mport uint16
		var mcode uint32
		var mnode *Node

		// Byte 00 Member status byte
		mstatus = NodeStatus(bytes[p])
		p++

		// Bytes 01-04 member IP
		if bytes[p] > 0 {
			mip = net.IPv4(
				bytes[p+0],
				bytes[p+1],
				bytes[p+2],
				bytes[p+3]).To4()
		}
		p += 4

		// Bytes 05-06 member response port
		mport, p = decodeUint16(bytes, p)

		// Bytes 07-10 member message code
		mcode, p = decodeUint32(bytes, p)

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
