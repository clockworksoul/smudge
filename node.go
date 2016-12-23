/*
Copyright 2016 The Blackfish Authors.

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

package blackfish

import (
	"fmt"
	"net"
	"time"
)

// Node represents a single node in the cluster.
type Node struct {
	ip               net.IP
	port             uint16
	timestamp        uint32
	address          string
	status           NodeStatus
	broadcastCounter int8
}

// Address rReturns the address for this node in string format, which is simply
// the node's local IP and listen port. This is used as a unique identifier
// throughout the code base.
func (n *Node) Address() string {
	if n.address == "" {
		n.address = fmt.Sprintf("%s:%d", n.ip.String(), n.port)
	}

	return n.address
}

// Age returns the time since we last heard from this node, in milliseconds.
func (n *Node) Age() uint32 {
	return GetNowInMillis() - n.timestamp
}

// BroadcastCounter returns the number of times remaining that current status
// will be broadcast by this node to other nodes.
func (n *Node) BroadcastCounter() int8 {
	return n.broadcastCounter
}

// IP returns the IP associated with this node.
func (n *Node) IP() net.IP {
	return n.ip
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

// GetNowInMillis returns the current local time in milliseconds since the
// epoch.
func GetNowInMillis() uint32 {
	return uint32(time.Now().UnixNano() / int64(time.Millisecond))
}
