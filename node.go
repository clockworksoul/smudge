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
	"fmt"
	"net"
	"time"
)

const (
	// PingNoData is returned by n.PingMillis() to indicate that a node has
	// not yet been pinged, and therefore no ping data exists.
	PingNoData int = -1

	// PingTimedOut is returned by n.PingMillis() to indicate that a node's
	// last PING timed out. This is the typical value for dead nodes.
	PingTimedOut int = -2
)

// Node represents a single node in the cluster.
type Node struct {
	ip          net.IP
	port        uint16
	timestamp   uint32
	address     string
	pingMillis  int
	status      NodeStatus
	emitCounter int8
	heartbeat   uint32
}

// Address rReturns the address for this node in string format, which is simply
// the node's local IP and listen port. This is used as a unique identifier
// throughout the code base.
func (n *Node) Address() string {
	if n.address == "" {
		n.address = nodeAddressString(n.ip, n.port)
	}

	return n.address
}

// Age returns the time since we last heard from this node, in milliseconds.
func (n *Node) Age() uint32 {
	return GetNowInMillis() - n.timestamp
}

// EmitCounter returns the number of times remaining that current status
// will be emitted by this node to other nodes.
func (n *Node) EmitCounter() int8 {
	return n.emitCounter
}

// IP returns the IP associated with this node.
func (n *Node) IP() net.IP {
	return n.ip
}

// PingMillis returns the milliseconds transpired between the most recent
// PING to this node and its responded ACK. If this node has not yet been
// pinged, this vaue will be PingNoData (-1). If this node's last PING timed
// out, this value will be PingTimedOut (-2).
func (n *Node) PingMillis() int {
	return n.pingMillis
}

// Port returns the port associated with this node.
func (n *Node) Port() uint16 {
	return n.port
}

// Status returns this node's current status.
func (n *Node) Status() NodeStatus {
	return n.status
}

// Timestamp returns the timestamp of this node's last ping or status update,
// in milliseconds from the epoch
func (n *Node) Timestamp() uint32 {
	return n.timestamp
}

// Touch updates the timestamp to the local time in milliseconds.
func (n *Node) Touch() {
	n.timestamp = GetNowInMillis()
}

func nodeAddressString(ip net.IP, port uint16) string {
	if ip.To4() != nil {
		return fmt.Sprintf("%s:%d", ip.String(), port)
	}
	return fmt.Sprintf("[%s]:%d", ip.String(), port)
}

// GetNowInMillis returns the current local time in milliseconds since the
// epoch.
func GetNowInMillis() uint32 {
	return uint32(time.Now().UnixNano() / int64(time.Millisecond))
}
