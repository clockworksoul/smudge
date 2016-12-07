package fleacircus

import "fmt"

type Node struct {
	Host       string
	Port       uint16
	Heartbeats uint32
	Timestamp  uint32
	address    string
}

/**
 * The time since we last heard from this node, in milliseconds.
 */
func (n *Node) Age() uint32 {
	return GetNowInMillis() - n.Timestamp
}

func (n *Node) Address() string {
	if n.address == "" {
		n.address = fmt.Sprintf("%s:%d", n.Host, n.Port)
	}

	return n.address
}

/**
 * Updates the timestamp to the local time in nanos
 */
func (n *Node) Touch() {
	n.Timestamp = GetNowInMillis()
}
