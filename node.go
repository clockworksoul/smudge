package blackfish

import (
	"fmt"
	"net"
	"time"
)

// Represents a single node in the cluster.
type Node struct {
	ip               net.IP
	port             uint16
	heartbeats       uint32
	timestamp        uint32
	address          string
	status           NodeStatus
	broadcastCounter byte
}

func (n *Node) Address() string {
	if n.address == "" {
		n.address = fmt.Sprintf("%s:%d", n.ip.String(), n.port)
	}

	return n.address
}

// The time since we last heard from this node, in milliseconds.
func (n *Node) Age() uint32 {
	return GetNowInMillis() - n.timestamp
}

// The number of times the current status will be broadcast to other nodes.
func (n *Node) BroadcastCounter() byte {
	return n.broadcastCounter
}

// The node's current heartbeat counter.
// TODO Do we still need this now?
func (n *Node) Heartbeats() uint32 {
	return n.heartbeats
}

// The IP associated with this node.
func (n *Node) IP() net.IP {
	return n.ip
}

// The port associated with this node.
func (n *Node) Port() uint16 {
	return n.port
}

// This node's current status.
func (n *Node) Status() NodeStatus {
	return n.status
}

// The timestamp of this node's last ping or status update, in milliseconds
// from the epoch
func (n *Node) Timestamp() uint32 {
	return n.timestamp
}

// Updates the timestamp to the local time in milliseconds.
func (n *Node) Touch() {
	n.timestamp = GetNowInMillis()
}

func GetNowInMillis() uint32 {
	return uint32(time.Now().UnixNano() / int64(time.Millisecond))
}
