package blackfish

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

var current_heartbeat uint32

var pending_acks = struct {
	sync.RWMutex
	m map[string]*pendingAck
}{m: make(map[string]*pendingAck)}

var this_host_address string

var this_host *Node

var TIMEOUT_MILLIS uint32 = 2500

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

		this_host_address = me.Address()
		this_host = &me

		fmt.Println("My host address:", this_host_address)
		fmt.Println("My host:", this_host)
	}

	go ListenUDP(GetListenPort())

	go startTimeoutCheckLoop()

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
		buf := make([]byte, 17)
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
	fmt.Println(live_nodes.Length(), "nodes")

	live_nodes.RLock()
	for _, node := range live_nodes.nodes {
		go PingNode(node)
	}
	live_nodes.RUnlock()
}

// Initiates a ping of `count` nodes. Passing 0 is equivalent to calling
// PingAllNodes().
func PingNNodes(count int) {
	rnodes := GetRandomNodes(count)

	// Loop over nodes and ping them
	live_nodes.RLock()
	for _, node := range *rnodes {
		go PingNode(&node)
	}
	live_nodes.RUnlock()
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

func receiveMessageUDP(addr *net.UDPAddr, msg_bytes []byte) error {
	msg, err := decodeMessage(addr, msg_bytes)
	if err != nil {
		return err
	}

	// If the sender is new to us (and it isn't this host), we know it now.
	if msg.sender == nil {
		sender := Node{
			Host:      msg.senderIP,
			Port:      msg.senderPort,
			Timestamp: GetNowInMillis()}

		if sender.Address() != this_host_address {
			_, msg.sender, _ = live_nodes.Add(sender)
		}
	}

	// Handle the verb. Each verb is three characters, and is one of the
	// following:
	//   PNG - Ping
	//   ACK - Acknowledge
	//   FWD - Forwarding ping (contains origin address)
	//   NFP - Non-forwarding ping
	switch {
	case msg.verb == "PING":
		err = receiveVerbPingUDP(msg)
	case msg.verb == "ACK":
		err = receiveVerbAckUDP(msg)
	case msg.verb == "FORWARD":
		err = receiveVerbForwardUDP(msg)
	case msg.verb == "NFPING":
		err = receiveVerbNonForwardPingUDP(msg)
	}

	if err != nil {
		return err
	}

	// Synchronize heartbeats
	if msg.senderCode > current_heartbeat {
		current_heartbeat = msg.senderCode - 1
	}

	return nil
}

func receiveVerbAckUDP(msg message) error {
	fmt.Println("GOT ACK FROM", msg.sender.Address(), msg.senderCode)

	key := msg.sender.Address() + ":" + strconv.FormatInt(int64(msg.senderCode), 10)

	pending_acks.RLock()
	_, ok := pending_acks.m[key]
	pending_acks.RUnlock()

	if ok {
		// TODO Keep statistics on response times

		msg.sender.Heartbeats = current_heartbeat
		msg.sender.Touch()

		pending_acks.Lock()

		// If this was a forwarded ping, respond to the callback node
		if pack, ok := pending_acks.m[key]; ok {
			if pack.Callback != nil {
				go transmitVerbAckUDP(pack.Callback, pack.CallbackCode)
			}
		}

		delete(pending_acks.m, key)
		pending_acks.Unlock()
	} else {
		fmt.Println("**NO", key)
	}

	return nil
}

func receiveVerbForwardUDP(msg message) error {
	fmt.Println("GOT FORWARD FROM", msg.sender.Address(), msg.senderCode)

	node := msg.downstream
	code := msg.downstreamCode
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)
	pack := pendingAck{Node: node,
		StartTime:    GetNowInMillis(),
		Callback:     node,
		CallbackCode: code}

	pending_acks.Lock()
	pending_acks.m[key] = &pack
	pending_acks.Unlock()

	return transmitVerbGenericUDP(node, nil, "NFPING", code)
}

func receiveVerbPingUDP(msg message) error {
	fmt.Println("GOT PING FROM", msg.sender.Address(), msg.senderCode)

	return transmitVerbAckUDP(msg.sender, msg.senderCode)
}

func receiveVerbNonForwardPingUDP(msg message) error {
	fmt.Println("GOT NFPING FROM", msg.sender.Address(), msg.senderCode)

	return transmitVerbAckUDP(msg.sender, msg.senderCode)
}

func startTimeoutCheckLoop() {
	for {
		pending_acks.Lock()
		for k, ack := range pending_acks.m {
			elapsed := ack.Elapsed()

			if elapsed > TIMEOUT_MILLIS {
				fmt.Println(k, "timed out after", TIMEOUT_MILLIS, " milliseconds")

				// If a pending ack has a "downstream" field defined, then
				// it's the result of a NFP and we don't forward it. If it
				// isn't defined, we forward this request to a random host.
				go doForwardOnTimeout(ack)

				delete(pending_acks.m, k)
			}
		}
		pending_acks.Unlock()

		time.Sleep(time.Millisecond * 1000)
	}
}

func doForwardOnTimeout(pack *pendingAck) {
	timed_out_address := pack.Node.Address()

	random_node := live_nodes.GetRandom(timed_out_address, this_host_address)

	if random_node == nil {
		fmt.Println(timed_out_address, "Cannot forward ping request: no more nodes")
	} else {
		fmt.Printf("Requesting indirect ping of %s via %s\n",
			pack.Node.Address(),
			random_node.Address())

		transmitVerbForwardUDP(random_node, pack.Node, current_heartbeat)
	}
}

func transmitVerbGenericUDP(node *Node, downstream *Node, verb string, code uint32) error {
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

	msg := message{
		verb:       verb,
		sender:     this_host,
		senderIP:   this_host.Host,
		senderPort: this_host.Port,
		senderCode: code}

	if downstream != nil {
		msg.downstream = downstream
		msg.downstreamIP = downstream.Host
		msg.downstreamPort = downstream.Port
		msg.downstreamCode = code
	}

	_, err = c.Write(encodeMessage(msg))
	if err != nil {
		return err
	}

	return nil
}

func transmitVerbForwardUDP(node *Node, downstream *Node, code uint32) error {
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)
	pack := pendingAck{Node: node, StartTime: GetNowInMillis(), Callback: downstream}

	pending_acks.Lock()
	pending_acks.m[key] = &pack
	pending_acks.Unlock()

	return transmitVerbGenericUDP(node, downstream, "FORWARD", code)
}

func transmitVerbAckUDP(node *Node, code uint32) error {
	return transmitVerbGenericUDP(node, nil, "ACK", code)
}

func transmitVerbPingUDP(node *Node, code uint32) error {
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)
	pack := pendingAck{Node: node, StartTime: GetNowInMillis()}

	pending_acks.Lock()
	pending_acks.m[key] = &pack
	pending_acks.Unlock()

	return transmitVerbGenericUDP(node, nil, "PING", code)
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
	// For now, the only supported verb is PNG; later we'll support FORWARD
	// and NFPNG ("non-forwarding ping") for a full SWIM implementation.

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
			case verb == "PNG":
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
	mergedNodes := live_nodes.mergeNodeLists(msgNodes)

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
	err = encoder.Encode("PNG")
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
		return errors.New("unexpected response on PNG: " + ack)
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

	live_nodes.mergeNodeLists(msgNodes)

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
	StartTime    uint32
	Node         *Node
	Callback     *Node
	CallbackCode uint32
}

func (a *pendingAck) Elapsed() uint32 {
	return GetNowInMillis() - a.StartTime
}
