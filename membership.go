package blackfish

import (
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
			IP:         ip,
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

	// Handle the verb. Each verb is three characters, and is one of the
	// following:
	//   PNG - Ping
	//   ACK - Acknowledge
	//   FWD - Forwarding ping (contains origin address)
	//   NFP - Non-forwarding ping
	switch msg.getVerb() {
	case VERB_PING:
		err = receiveVerbPingUDP(msg)
	case VERB_ACK:
		err = receiveVerbAckUDP(msg)
	case VERB_FORWARD:
		err = receiveVerbForwardUDP(msg)
	case VERB_NFPING:
		err = receiveVerbNonForwardPingUDP(msg)
	}

	if err != nil {
		return err
	}

	// Synchronize heartbeats
	if msg.getSenderCode() > current_heartbeat {
		current_heartbeat = msg.getSenderCode() - 1
	}

	return nil
}

func receiveVerbAckUDP(msg message) error {
	fmt.Println("GOT ACK FROM", msg.getSender().Address(), msg.getSenderCode())

	key := msg.getSender().Address() + ":" + strconv.FormatInt(int64(msg.getSenderCode()), 10)

	pending_acks.RLock()
	_, ok := pending_acks.m[key]
	pending_acks.RUnlock()

	if ok {
		// TODO Keep statistics on response times

		msg.getSender().Heartbeats = current_heartbeat
		msg.getSender().Touch()

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
	fmt.Println("GOT FORWARD FROM", msg.getSender().Address(), msg.getSenderCode())

	member := msg.getForwardTo()
	if member == nil {
		return errors.New("No forward-to member in message")
	}

	node := member.node
	code := member.code
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)

	pack := pendingAck{Node: node,
		StartTime:    GetNowInMillis(),
		Callback:     node,
		CallbackCode: code}

	pending_acks.Lock()
	pending_acks.m[key] = &pack
	pending_acks.Unlock()

	return transmitVerbGenericUDP(node, nil, VERB_NFPING, code)
}

func receiveVerbPingUDP(msg message) error {
	fmt.Println("GOT PING FROM", msg.getSender().Address(), msg.getSenderCode())

	return transmitVerbAckUDP(msg.getSender(), msg.getSenderCode())
}

func receiveVerbNonForwardPingUDP(msg message) error {
	fmt.Println("GOT NFPING FROM", msg.getSender().Address(), msg.getSenderCode())

	return transmitVerbAckUDP(msg.getSender(), msg.getSenderCode())
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

func transmitVerbGenericUDP(node *Node, forward_to *Node, verb byte, code uint32) error {
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
		_verb:       verb,
		_sender:     this_host,
		_senderCode: code}

	if forward_to != nil {
		msg.addMember(STATUS_FORWARD_TO, forward_to, code)
	}

	_, err = c.Write(msg.encode())
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

	return transmitVerbGenericUDP(node, downstream, VERB_FORWARD, code)
}

func transmitVerbAckUDP(node *Node, code uint32) error {
	return transmitVerbGenericUDP(node, nil, VERB_ACK, code)
}

func transmitVerbPingUDP(node *Node, code uint32) error {
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)
	pack := pendingAck{Node: node, StartTime: GetNowInMillis()}

	pending_acks.Lock()
	pending_acks.m[key] = &pack
	pending_acks.Unlock()

	return transmitVerbGenericUDP(node, nil, VERB_PING, code)
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
