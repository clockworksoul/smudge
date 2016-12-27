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

	// An index pointer
	p := 0

	if (len(bytes)-11)%11 != 0 {
		logfWarn("Inconsistent byte count received from %v: %d MOD 11 = %d\n",
			addr.IP,
			len(bytes),
			len(bytes)%11)

		return newMessage(255, nil, 0),
			errors.New("unexpected byte length received in message")
	}

	checksumStated, p := decodeUint32(bytes, p)
	checksumCalculated := adler32.Checksum(bytes[4:])
	if checksumCalculated != checksumStated {
		return newMessage(255, nil, 0),
			errors.New("checksum failure from " + addr.IP.String())
	}

	// Bytes 00    Verb (one of {P|A|F|N})
	verb := messageVerb(bytes[p])
	p++

	// Bytes 01-02 Sender response port
	senderPort, p := decodeUint16(bytes, p)

	// Bytes 03-06 Sender ID Code
	senderCode, p := decodeUint32(bytes, p)

	// Now that we have the IP and port, we can find the Node.
	sender := knownNodes.getByIP(addr.IP.To4(), senderPort)

	// We don't know this node, so create a new one!
	if sender == nil {
		sender, _ = CreateNodeByIP(addr.IP.To4(), senderPort)
	}

	// Now that we have the verb, node, and code, we can build the mesage
	m := newMessage(verb, sender, senderCode)

	if len(bytes) > p {
		m.members = decodeMembers(bytes[p:])
	}

	return m, nil
}

func decodeMembers(bytes []byte) []*messageMember {
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

// Convenience function. Creates a new message instance.
func newMessage(verb messageVerb, sender *Node, code uint32) message {
	return message{
		sender:     sender,
		senderCode: code,
		verb:       verb,
	}
}

type message struct {
	sender     *Node
	senderCode uint32
	verb       messageVerb
	members    []*messageMember
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
	size := 11 + (len(m.members) * 11)
	bytes := make([]byte, size, size)

	// An index pointer (start at 4 to accommodate checksum)
	p := 4

	// Bytes 00    Verb (one of {P|A|F|N})
	// Translation: the first character of the message verb
	bytes[p] = byte(m.verb)
	p++

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

// Represents a "member" of a message; i.e., a node that the sender knows
// about, about which it wishes to notify the downstream recipient.
type messageMember struct {
	code   uint32
	node   *Node
	status NodeStatus
}
