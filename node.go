package blackfish

import (
	"fmt"
	"net"
	"time"
)

type Node struct {
	Host       net.IP
	Port       uint16
	Heartbeats uint32
	Timestamp  uint32
	address    string
}

// The time since we last heard from this node, in milliseconds.
func (n *Node) Age() uint32 {
	return GetNowInMillis() - n.Timestamp
}

func (n *Node) Address() string {
	if n.address == "" {
		n.address = fmt.Sprintf("%s:%d", n.Host.String(), n.Port)
	}

	return n.address
}

// Updates the timestamp to the local time in nanos
func (n *Node) Touch() {
	n.Timestamp = GetNowInMillis()
}

func GetNowInMillis() uint32 {
	return uint32(time.Now().UnixNano() / int64(time.Millisecond))
}
