package blackfish

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

var current_heartbeat uint32

var pending_acks map[string]*pendingAck

var this_host_address string

func init() {
	pending_acks = make(map[string]*pendingAck)
}

func Begin() {
	// Add this host.
	ip, err := GetLocalIP()
	if err != nil {
		fmt.Println("Warning: Could not resolve host IP")
	} else {
		me := Node{
			Host:       ip,
			Port:       uint16(GetListenPort()),
			Heartbeats: current_heartbeat,
			Timestamp:  GetNowInMillis()}

		registerNewNode(me)

		this_host_address = me.Address()

		fmt.Println("My host address:", this_host_address)
	}

	go ListenUDP(GetListenPort())

	for {
		current_heartbeat++

		PruneDeadFromList()

		PingAllNodes()

		// 1 heartbeat in 10, we resurrect a random dead node
		if current_heartbeat%25 == 0 {
			ResurrectDeadNode()
		}

		time.Sleep(time.Millisecond * time.Duration(GetHeartbeatMillis()))
	}
}

func ListenUDP(port int) error {
	listenAddress, err := net.ResolveUDPAddr("udp", ":"+strconv.FormatInt(int64(port), 10))
	if err != nil {
		return err
	}

	/* Now listen at selected port */
	c, err := net.ListenUDP("udp", listenAddress)
	if err != nil {
		return err
	}
	defer c.Close()

	for {
		buf := make([]byte, 16)
		n, addr, err := c.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("UDP read error: ", err)
		}

		go func(addr *net.UDPAddr, msg []byte) {
	        err = receiveMessageUDP(addr, buf[0:n])
			if err != nil {
				fmt.Println(err)
			}
	    }(addr, buf[0:n])
	}
}

func PingAllNodes() {
	fmt.Println(len(live_nodes), "nodes")

	for _, node := range live_nodes {
		go PingNode(node)
	}
}

// Initiates a ping of `count` nodes. Passing 0 is equivalent to calling
// PingAllNodes().
func PingNNodes(count int) {
	rnodes := GetRandomNodes(count)

	// Loop over nodes and ping them
	for _, node := range *rnodes {
		go PingNode(&node)
	}
}

// User-friendly method to explicitly ping a node. Calls the low-level
// doPingNode(), and outputs a message if it fails.
func PingNode(node *Node) error {
	err := transmitVerbPingUDP(node, current_heartbeat)
	if err != nil {
		fmt.Println("Failure to ping", node, "->", err)
	}

	return err
}

func constructUDPMessage(verb string, code uint32) []byte {
	bytes_out := []byte(verb)

	bytes_out = append(bytes_out, byte(0))

	for i := uint32(0); i < 32; i += 8 {
		code >>= i
		bytes_out = append(bytes_out, byte(code))
	}

	return bytes_out
}

func receiveMessageUDP(addr *net.UDPAddr, msg []byte) error {
	verb, code, err := parseUDPMessage(msg)
	if err != nil {
		return err
	}

	// GET THE NODE
	node := lookupNodeByAddress(addr.IP, 0)
	if node == nil {
		fmt.Println("Unrecognized IP:", addr.IP)
	} else {
		// Handle the verb
		//
		switch {
		case verb == "PING":
			err = receiveVerbPingUDP(node, code)
		case verb == "ACK":
			err = receiveVerbAckUDP(node, code)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func receiveVerbPingUDP(node *Node, code uint32) error {
	fmt.Println("GOT PING FROM", node.Address(), code)

	return transmitVerbAckUDP(node, code)
}

func receiveVerbAckUDP(node *Node, code uint32) error {
	fmt.Println("GOT ACK FROM", node.Address(), code)

	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)

	if _, ok := pending_acks[key]; ok {
		// now := GetNowInMillis()
		// start := ack.StartTime
		// elapsed := now - start

		node.Heartbeats = current_heartbeat
		node.Touch()

		delete(pending_acks, key)
	} else {
		fmt.Println("**NO", key)
	}

	return nil
}

func transmitVerbGenericUDP(node *Node, verb string, code uint32) error {
	// Transmit the ACK
	remote_addr, err := net.ResolveUDPAddr("udp", node.Address())
	if err != nil {
		return err
	}

	c, err := net.DialUDP("udp", nil, remote_addr)
	if err != nil {
		return err
	}
	defer c.Close()

	_, err = c.Write(constructUDPMessage(verb, code))
	if err != nil {
		return err
	}

	return nil
}

func transmitVerbAckUDP(node *Node, code uint32) error {
	return transmitVerbGenericUDP(node, "ACK", code)
}

func transmitVerbPingUDP(node *Node, code uint32) error {
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)
	pack := pendingAck{Node: node, StartTime: GetNowInMillis()}
	pending_acks[key] = &pack

	return transmitVerbGenericUDP(node, "PING", code)
}

func parseUDPMessage(msg_bytes []byte) (string, uint32, error) {
	var verb string
	var code uint32

	// Scan to find the null byte and use that to find the verb
	for i, b := range msg_bytes {
		if b == 0 {
			verb = string(msg_bytes[:i])
			break
		}
	}

	// Everything after is the code value
	for j := len(msg_bytes) - 1; j > len(verb); j-- {
		code <<= 8
		code |= uint32(msg_bytes[j])
	}

	if verb == "" {
		return verb, code, errors.New("Verb not found")
	}

	return verb, code, nil
}

///////////////////////////////////////////////////////////////////////////////
/// ATTIC IS BELOW
///////////////////////////////////////////////////////////////////////////////

func doPingNodeTCP(node *Node) error {
	// TODO DON'T USE TCP. Switch to UDP, or better still, raw sockets.
	c, err := net.Dial("tcp", node.Address())
	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(c)
	decoder := gob.NewDecoder(c)

	// err = transmitVerbPing(&c, encoder, decoder)
	// if err != nil {
	// 	return err
	// }

	err = transmitVerbList(&c, encoder, decoder)
	if err != nil {
		return err
	}

	node.Heartbeats = current_heartbeat
	node.Touch()

	c.Close()

	return nil
}

func handleMembershipPing(c *net.Conn) {
	// var msgNodes *[]Node
	var verb string
	var err error

	// Every ping comes in two parts: the verb and the node list.
	// For now, the only supported verb is PING; later we'll support FORWARD
	// and NFPING ("non-forwarding ping") for a full SWIM implementation.

	decoder := gob.NewDecoder(*c)
	encoder := gob.NewEncoder(*c)

Loop:
	for {
		// First, receive the verb
		//
		derr := decoder.Decode(&verb)
		if derr != nil {
			break Loop
		} else {
			// Handle the verb
			//
			switch {
			case verb == "PING":
				err = receiveVerbPing(c, encoder, decoder)
			case verb == "LIST":
				err = receiveVerbList(c, encoder, decoder)
			}

			if err != nil {
				fmt.Println("Error receiving verb:", err)
				break Loop
			}
		}
	}

	(*c).Close()
}

func receiveNodes(decoder *gob.Decoder) (*[]Node, error) {
	var mnodes []Node

	// Second, receive the list
	//
	var length int
	var host net.IP
	var port uint16
	var heartbeats uint32
	var err error

	err = decoder.Decode(&length)
	if err != nil {
		fmt.Println("Error receiving list:", err)
		return &mnodes, err
	}

	for i := 0; i < length; i++ {
		err = decoder.Decode(&host)
		if err != nil {
			fmt.Println("Error receiving list (host):", err)
			return &mnodes, err
		}

		err = decoder.Decode(&port)
		if err != nil {
			fmt.Println("Error receiving list (port):", err)
			return &mnodes, err
		}

		err = decoder.Decode(&heartbeats)
		if err != nil {
			fmt.Println("Error receiving list (heartbeats):", err)
			return &mnodes, err
		}

		newNode := Node{
			Host:       host,
			Port:       port,
			Heartbeats: heartbeats,
			Timestamp:  GetNowInMillis()}

		mnodes = append(mnodes, newNode)

		// Does this node have a higher heartbeat than our current one?
		// If so, synchronize heartbeats.
		//
		if heartbeats > current_heartbeat {
			current_heartbeat = heartbeats
		}
	}

	return &mnodes, err
}

func receiveVerbPing(c *net.Conn, encoder *gob.Encoder, decoder *gob.Decoder) error {
	return encoder.Encode("ACK")
}

func receiveVerbList(c *net.Conn, encoder *gob.Encoder, decoder *gob.Decoder) error {
	var msgNodes *[]Node
	var err error

	// Receive the entire node list from the peer, but don't merge it yet!
	//
	msgNodes, err = receiveNodes(decoder)
	if err != nil {
		return err
	}

	// Finally, merge the list of nodes we received from the peer into ours
	//
	mergedNodes := mergeNodeLists(msgNodes)

	// Reply with our own nodes list
	//
	err = transmitNodes(encoder, getRandomNodes(GetMaxNodesToTransmit(), mergedNodes))
	if err != nil {
		return err
	}

	return nil
}

func transmitNodes(encoder *gob.Encoder, mnodes *[]Node) error {
	var err error

	// Send the length
	//
	err = encoder.Encode(len(*mnodes))
	if err != nil {
		return err
	}

	for _, n := range *mnodes {
		err = encoder.Encode(n.Host)
		if err != nil {
			return err
		}

		err = encoder.Encode(n.Port)
		if err != nil {
			return err
		}

		err = encoder.Encode(n.Heartbeats)
		if err != nil {
			return err
		}
	}

	return nil
}

func transmitVerbPing(c *net.Conn, encoder *gob.Encoder, decoder *gob.Decoder) error {
	var err error
	var ack string

	// Send the verb
	//
	err = encoder.Encode("PING")
	if err != nil {
		return err
	}

	// Receive the response
	//
	err = decoder.Decode(&ack)
	if err != nil {
		return err
	}

	if ack != "ACK" {
		return errors.New("unexpected response on PING: " + ack)
	}

	return nil
}

func transmitVerbList(c *net.Conn, encoder *gob.Encoder, decoder *gob.Decoder) error {
	var err error

	// Send the verb
	//
	err = encoder.Encode("LIST")
	if err != nil {
		return err
	}

	transmitNodes(encoder, GetRandomNodes(GetMaxNodesToTransmit()))

	msgNodes, err := receiveNodes(decoder)
	if err != nil {
		return err
	}

	mergeNodeLists(msgNodes)

	return nil
}

// Starts the server on the indicated node. This is a blocking operation,
// so you probably want to execute this as a gofunc.
func ListenTCP(port int) error {
	// TODO DON'T USE TCP. Switch to UDP, or better still, raw sockets.
	ln, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(port), 10))
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Println("Listening on port", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		// Handle the connection
		go handleMembershipPing(&conn)
	}

	return nil
}

type pendingAck struct {
	StartTime uint32
	Node      *Node
}

func (a *pendingAck) elapsed() uint32 {
	return GetNowInMillis() - a.StartTime
}
