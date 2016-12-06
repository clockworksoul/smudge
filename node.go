package fleacircus

import "fmt"

type Node struct {
	Host      string
	Port      uint16
	Timestamp uint32
	address   string
}

/**
 * The time since we last heard from this node, in milliseconds.
 */
func (this *Node) Age() uint32 {
	return GetNowInMillis() - this.Timestamp
}

func (this *Node) Address() string {
	if this.address == "" {
		this.address = fmt.Sprintf("%s:%d", this.Host, this.Port)
	}

	return this.address
}

/**
 * Updates the timestamp to the local time in nanos
 */
func (this *Node) Touch() {
	this.Timestamp = GetNowInMillis()
}
