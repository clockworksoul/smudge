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
	"fmt"
	"net"
	"sort"
	"sync"
)

const (
	// Emit counters for broadcasts can be less than 0. We transmit positive
	// numbers, and decrement all the others. At this value, the broadcast
	// is removed from the map all together. This ensures broadcasts are
	// emitted briefly, but retained long enough to not be received twice.
	broadcastRemoveValue int8 = int8(-100)
)

// The index counter value for the next broadcast message
var indexCounter uint32 = 1

// Emitted broadcasts. Once they are added here, the membership machinery will
// pick them up and piggyback them onto standard messages.
var broadcasts = struct {
	sync.RWMutex
	m map[string]*Broadcast
}{m: make(map[string]*Broadcast)}

// Broadcast represents a packet of bytes emitted across the cluster on top of
// the status update infrastructure. Although useful, its payload is limited
// to only 256 bytes.
type Broadcast struct {
	bytes       []byte
	origin      *Node
	index       uint32
	label       string
	emitCounter int8
}

// Bytes returns a copy of this broadcast's bytes. Manipulating the contents
// of this slice will not be reflected in the contents of the broadcast.
func (b *Broadcast) Bytes() []byte {
	length := len(b.bytes)
	bytesCopy := make([]byte, length, length)
	copy(bytesCopy, b.bytes)

	return bytesCopy
}

// Index returns the origin message index for this broadcast. This value is
// incremented for each broadcast. The combination of
// originIP:originPort:Index is unique.
func (b *Broadcast) Index() uint32 {
	return b.index
}

// Label returns a unique label string composed of originIP:originPort:Index.
func (b *Broadcast) Label() string {
	if b.label == "" {
		b.label = fmt.Sprintf("%s:%d:%d",
			b.origin.ip.String(),
			b.origin.port,
			b.index)
	}

	return b.label
}

// Origin returns the node that this broadcast originated from.
func (b *Broadcast) Origin() *Node {
	return b.origin
}

// BroadcastBytes allows a user to emit a short broadcast in the form of a byte
// slice, which will be transmitted at most once to all other healthy current
// members. Members that join after the broadcast has already propagated
// through the cluster will not receive the message. The maximum broadcast
// length is 256 bytes.
func BroadcastBytes(bytes []byte) error {
	if len(bytes) > GetMaxBroadcastBytes() {
		emsg := fmt.Sprintf(
			"broadcast payload length exceeds %d bytes",
			GetMaxBroadcastBytes())

		return errors.New(emsg)
	}

	broadcasts.Lock()

	bcast := Broadcast{
		origin:      thisHost,
		index:       indexCounter,
		bytes:       bytes,
		emitCounter: int8(emitCount())}

	broadcasts.m[bcast.Label()] = &bcast

	indexCounter++

	broadcasts.Unlock()

	return nil
}

// BroadcastString allows a user to emit a short broadcast in the form of a
// string, which will be transmitted at most once to all other healthy current
// members. Members that join after the broadcast has already propagated
// through the cluster will not receive the message. The maximum broadcast
// length is 256 bytes.
func BroadcastString(str string) error {
	return BroadcastBytes([]byte(str))
}

// Message contents for IPv6
// Bytes       Content
// ------------------------
// Bytes 00-15 Origin IP (00-03 for IPv4)
// Bytes 16-17 Origin response port (04-05 for IPv4)
// Bytes 18-21 Origin broadcast counter (06-09 for IPv4)
// Bytes 22-23 Payload length (bytes) (10-11 for IPv4)
// Bytes 24-NN Payload (12-NN for IPv4)
func (b *Broadcast) encode() []byte {
	size := 8 + ipLen + len(b.bytes)
	bytes := make([]byte, size, size)

	// Index pointer
	p := 0

	// Bytes 00-15: Origin IP
	ip := b.origin.IP()
	if ip.To4() != nil {
		ip = ip.To4()
	}

	for i := 0; i < ipLen; i++ {
		bytes[p+i] = ip[i]
	}
	p += ipLen

	// Bytes 16-17 Origin response port
	p += encodeUint16(b.origin.Port(), bytes, p)

	// Bytes 18-21 Origin broadcast counter
	p += encodeUint32(b.index, bytes, p)

	// Bytes 22-23 Payload length (bytes)
	p += encodeUint16(uint16(len(b.bytes)), bytes, p)

	// Bytes 24-NN Payload
	for i, by := range b.bytes {
		bytes[i+p] = by
	}

	return bytes
}

// Message contents
// Bytes       Content
// ------------------------
// Bytes 00-15 Origin IP (00-03 on IPv4)
// Bytes 16-17 Origin response port (04-05 on IPv4)
// Bytes 18-21 Origin broadcast counter (06-09 on IPv4)
// Bytes 22-23 Payload length (bytes) (10-11 on IPv4)
// Bytes 24-NN Payload (12-NN on IPv4)
func decodeBroadcast(bytes []byte) (*Broadcast, error) {
	var index uint32
	var port uint16
	var ip net.IP
	var length uint16

	// An index pointer
	p := 0

	if ipLen == net.IPv6len {
		// Bytes 00-15 Origin IP
		ip = make(net.IP, net.IPv6len)
		copy(ip, bytes[p:p+16])
	} else {
		// Bytes 00-03 Origin IPv4
		ip = net.IPv4(bytes[p+0], bytes[p+1], bytes[p+2], bytes[p+3])
	}

	p += ipLen

	// Bytes 16-17 Origin response port
	port, p = decodeUint16(bytes, p)

	// Bytes 18-21 Origin broadcast counter
	index, p = decodeUint32(bytes, p)

	// Bytes 22-23 Payload length (bytes)
	length, p = decodeUint16(bytes, p)

	// Now that we have the IP and port, we can find the Node.
	origin := knownNodes.getByIP(ip, port)

	// We don't know this node, so create a new one!
	if origin == nil {
		origin, _ = CreateNodeByIP(ip, port)
	}

	bcast := Broadcast{
		origin:      origin,
		index:       index,
		bytes:       bytes[p : p+int(length)],
		emitCounter: int8(emitCount())}

	err := checkOrigin(origin)
	if err != nil {
		logWarn(err)
		return &bcast, err
	}

	if int(length) > GetMaxBroadcastBytes() {
		return &bcast,
			errors.New("message length exceeds maximum length")
	}

	return &bcast, nil
}

// getBroadcastToEmit identifies the single known broadcast with the highest
// emitCounter value (which can be negative), and returns it. If multiple
// broadcasts have the same value, one is arbitrarily chosen.
func getBroadcastToEmit() *Broadcast {
	// Get all broadcast messages.
	values := make([]*Broadcast, 0, 0)
	broadcasts.RLock()
	for _, v := range broadcasts.m {
		values = append(values, v)
	}
	broadcasts.RUnlock()

	// Remove all overly-emitted messages from the list
	broadcastSlice := make([]*Broadcast, 0, 0)
	broadcasts.Lock()
	for _, b := range values {
		if b.emitCounter <= broadcastRemoveValue {
			logDebug("Removing", b.Label(), "from recently updated list")
			delete(broadcasts.m, b.Label())
		} else {
			broadcastSlice = append(broadcastSlice, b)
		}
	}
	broadcasts.Unlock()

	if len(broadcastSlice) > 0 {
		// Put the newest nodes on top.
		sort.Sort(byBroadcastEmitCounter(broadcastSlice))
		return broadcastSlice[0]
	}

	return nil
}

// receiveBroadcast is called by receiveMessageUDP when a broadcast payload
// is found in a message.
func receiveBroadcast(broadcast *Broadcast) {
	if broadcast == nil {
		return
	}

	err := checkOrigin(broadcast.Origin())
	if err != nil {
		logWarn(err)
		return
	}

	label := broadcast.Label()

	broadcasts.Lock()
	_, contains := broadcasts.m[label]
	if !contains {
		broadcasts.m[label] = broadcast
	}
	broadcasts.Unlock()

	if !contains {
		logfInfo("Broadcast [%s]=%s",
			label,
			string(broadcast.Bytes()))

		doBroadcastUpdate(broadcast)
	}
}

// checkBroadcastOrigin checks wether the origin is set correctly
func checkOrigin(origin *Node) error {
	// normalize to IPv4 or IPv6 to check below
	ip := origin.IP()
	if ip.To4() != nil {
		ip = ip.To4()
	}

	if (ip[0] == 0) || origin.Port() == 0 {
		return errors.New("Received originless broadcast")
	}
	return nil
}

// byBroadcastEmitCounter implements sort.Interface for []*Broadcast based on
// the emitCounter field.
type byBroadcastEmitCounter []*Broadcast

func (a byBroadcastEmitCounter) Len() int {
	return len(a)
}

func (a byBroadcastEmitCounter) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a byBroadcastEmitCounter) Less(i, j int) bool {
	return a[i].emitCounter > a[j].emitCounter
}
