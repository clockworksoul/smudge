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
	// The maximum byte length of a broadcast. This is constrained by the
	// maximum safe UDP packet size of 508 bytes, which must also contain
	// additional message overhead and status updates.
	MaxBroadcastBytes int = 256

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
	bcast := Broadcast{
		origin:      thisHost,
		index:       indexCounter,
		bytes:       bytes,
		emitCounter: int8(emitCount())}

	if len(bytes) > MaxBroadcastBytes {
		emsg := fmt.Sprintf(
			"broadcast payload length exceeds %d bytes",
			MaxBroadcastBytes)

		return errors.New(emsg)
	}

	broadcasts.Lock()
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

// Message contents
// Bytes       Content
// ------------------------
// Bytes 00-03 Origin IP
// Bytes 04-05 Origin response port
// Bytes 06-09 Origin broadcast counter
// Bytes 10-11 Payload length (bytes)
// Bytes 12-NN Payload
func (b *Broadcast) encode() []byte {
	size := 12 + len(b.bytes)
	bytes := make([]byte, size, size)

	// Index pointer
	p := 0

	// Bytes 00-03: Origin IP
	ip := b.origin.IP()
	for i := 0; i < 4; i++ {
		bytes[p+i] = ip[i]
	}
	p += 4

	// Bytes 04-05 Origin response port
	p += encodeUint16(b.origin.Port(), bytes, p)

	// Bytes 06-09 Origin broadcast counter
	p += encodeUint32(b.index, bytes, p)

	// Bytes 10-11 Payload length (bytes)
	p += encodeUint16(uint16(len(b.bytes)), bytes, p)

	// Bytes 12-NN Payload
	for i, by := range b.bytes {
		bytes[i+p] = by
	}

	if bytes[0] == 0 {
		panic("Sending empty broadcast")
	}

	return bytes
}

// Message contents
// Bytes       Content
// ------------------------
// Bytes 00-03 Origin IP
// Bytes 04-05 Origin response port
// Bytes 06-09 Origin broadcast counter
// Bytes 10-11 Payload length (bytes)
// Bytes 12-NN Payload
func decodeBroadcast(bytes []byte) (*Broadcast, error) {
	var index uint32
	var port uint16
	var ip net.IP
	var length uint16

	// An index pointer
	p := 0

	// Bytes 00-03 Origin IP
	ip = net.IPv4(
		bytes[p+0],
		bytes[p+1],
		bytes[p+2],
		bytes[p+3]).To4()
	p += 4

	// Bytes 04-05 Origin response port
	port, p = decodeUint16(bytes, p)

	// Bytes 06-09 Origin broadcast counter
	index, p = decodeUint32(bytes, p)

	// Bytes 10-11 Payload length (bytes)
	length, p = decodeUint16(bytes, p)

	// Now that we have the IP and port, we can find the Node.
	origin := knownNodes.getByIP(ip.To4(), port)

	// We don't know this node, so create a new one!
	if origin == nil {
		origin, _ = CreateNodeByIP(ip.To4(), port)
	}

	bcast := Broadcast{
		origin:      origin,
		index:       index,
		bytes:       bytes[p : p+int(length)],
		emitCounter: int8(emitCount())}

	if origin.IP()[0] == 0 || origin.Port() == 0 {
		logWarn("Received originless broadcast")

		return &bcast,
			errors.New("received originless broadcast")
	}

	if int(length) > MaxBroadcastBytes {
		return &bcast,
			errors.New("message length exceeds maximum length of 256 bytes")
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

	if broadcast.Origin().IP()[0] == 0 || broadcast.Origin().Port() == 0 {
		logWarn("Received originless broadcast")
		return
	}

	label := broadcast.Label()

	broadcasts.Lock()
	_, contains := broadcasts.m[label]
	broadcasts.Unlock()

	if !contains {
		broadcasts.Lock()
		broadcasts.m[label] = broadcast
		broadcasts.Unlock()

		logfInfo("Broadcast [%s]=%s\n",
			label,
			string(broadcast.Bytes()))

		doBroadcastUpdate(broadcast)
	}
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
