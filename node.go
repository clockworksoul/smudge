package blackfish

import (
	"fmt"
	"net"
	"time"
)

const (
	STATUS_JOINED     = byte(0x00)
	STATUS_ALIVE      = byte(0x01)
	STATUS_FORWARD_TO = byte(0x02)
	STATUS_LEFT       = byte(0x03)
	STATUS_DIED       = byte(0x04)
	STATUS_SUSPECTED  = byte(0x05)
)

type Node struct {
	IP                net.IP
	Port              uint16
	Heartbeats        uint32
	Timestamp         uint32
	address           string
	status            byte
	broadcast_counter byte
}

func (n *Node) StatusString() string {
	var str string

	switch n.status {
	case STATUS_FORWARD_TO:
		str = "FORWARD_TO"
	case STATUS_JOINED:
		str = "JOINED"
	case STATUS_LEFT:
		str = "LEFT"
	case STATUS_DIED:
		str = "DIED"
	case STATUS_SUSPECTED:
		str = "SUSPECTED"
	case STATUS_ALIVE:
		str = "ALIVE"
	default:
		str = "INITIAL"
	}

	return str
}

// The time since we last heard from this node, in milliseconds.
func (n *Node) Age() uint32 {
	return GetNowInMillis() - n.Timestamp
}

func (n *Node) Address() string {
	if n.address == "" {
		n.address = fmt.Sprintf("%s:%d", n.IP.String(), n.Port)
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
