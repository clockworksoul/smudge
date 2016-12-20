package blackfish

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

var currentHeartbeat uint32

var pendingAcks = struct {
	sync.RWMutex
	m map[string]*pendingAck
}{m: make(map[string]*pendingAck)}

var thisHostAddress string

var thisHost *Node

var TIMEOUT_MILLIS uint32 = 150 // TODO Calculate this as the 99th percentile?

/******************************************************************************
 * Exported functions (for public consumption)
 *****************************************************************************/

func Begin() {
	// Add this host.
	ip, err := GetLocalIP()
	if err != nil {
		fmt.Println("Warning: Could not resolve host IP")
	} else {
		me := Node{
			IP:         ip,
			Port:       uint16(GetListenPort()),
			Heartbeats: currentHeartbeat,
			Timestamp:  GetNowInMillis()}

		thisHostAddress = me.Address()
		thisHost = &me

		fmt.Println("My host address:", thisHostAddress)
		fmt.Println("My host:", thisHost)
	}

	// Add this node's status. Don't update any other node's statuses: they'll
	// report those back to us.
	AddNode(thisHost)
	UpdateNodeStatus(thisHost, STATUS_JOINED)

	go listenUDP(GetListenPort())

	go startTimeoutCheckLoop()

	for {
		currentHeartbeat++

		fmt.Printf("[%d] %d hosts\n", currentHeartbeat, liveNodes.length())
		// PruneDeadFromList()

		// Ping one random node
		node := liveNodes.getRandom(thisHostAddress)
		if node != nil {
			PingNode(node)
		} else {
			fmt.Println("No nodes to ping. :(")
		}

		// PingAllNodes()

		// 1 heartbeat in 10, we resurrect a random dead node
		// if currentHeartbeat%25 == 0 {
		// 	ResurrectDeadNode()
		// }

		time.Sleep(time.Millisecond * time.Duration(GetHeartbeatMillis()))
	}
}

func PingAllNodes() {
	fmt.Println(liveNodes.length(), "nodes")

	liveNodes.RLock()
	for _, node := range liveNodes.nodes {
		go PingNode(node)
	}
	liveNodes.RUnlock()
}

// Initiates a ping of `count` nodes. Passing 0 is equivalent to calling
// PingAllNodes().
func PingNNodes(count int) {
	rnodes := liveNodes.getRandomNodes(count)

	// Loop over nodes and ping them
	liveNodes.RLock()
	for _, node := range rnodes {
		go PingNode(node)
	}
	liveNodes.RUnlock()
}

// User-friendly method to explicitly ping a node. Calls the low-level
// doPingNode(), and outputs a message if it fails.
func PingNode(node *Node) error {
	err := transmitVerbPingUDP(node, currentHeartbeat)
	if err != nil {
		fmt.Println("Failure to ping", node, "->", err)
	}

	return err
}

/******************************************************************************
 * Private functions (for internal use only)
 *****************************************************************************/

func listenUDP(port int) error {
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
		buf := make([]byte, 512)
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

func receiveMessageUDP(addr *net.UDPAddr, msg_bytes []byte) error {
	msg, err := decodeMessage(addr, msg_bytes)
	if err != nil {
		return err
	}

	fmt.Printf("GOT %s FROM %v code=%d\n",
		getVerbString(msg.verb),
		msg.sender.Address(),
		msg.senderCode)

	updateStatusesFromMessage(msg)

	// Handle the verb. Each verb is three characters, and is one of the
	// following:
	//   PNG - Ping
	//   ACK - Acknowledge
	//   FWD - Forwarding ping (contains origin address)
	//   NFP - Non-forwarding ping
	switch msg.verb {
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
	if msg.senderCode > currentHeartbeat {
		currentHeartbeat = msg.senderCode - 1
	}

	return nil
}

func receiveVerbAckUDP(msg message) error {
	key := msg.sender.Address() + ":" + strconv.FormatInt(int64(msg.senderCode), 10)

	pendingAcks.RLock()
	_, ok := pendingAcks.m[key]
	pendingAcks.RUnlock()

	if ok {
		// TODO Keep statistics on response times

		msg.sender.Heartbeats = currentHeartbeat
		msg.sender.Touch()

		pendingAcks.Lock()

		// If this was a forwarded ping, respond to the callback node
		if pack, ok := pendingAcks.m[key]; ok {
			if pack.Callback != nil {
				go transmitVerbAckUDP(pack.Callback, pack.CallbackCode)
			}
		}

		delete(pendingAcks.m, key)
		pendingAcks.Unlock()
	}

	return nil
}

func receiveVerbForwardUDP(msg message) error {
	if len(msg.members) >= 0 &&
		msg.members[0].status == STATUS_FORWARD_TO {
		member := msg.members[0]
		node := member.node
		code := member.code
		key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)

		pack := pendingAck{Node: node,
			StartTime:    GetNowInMillis(),
			Callback:     node,
			CallbackCode: code}

		pendingAcks.Lock()
		pendingAcks.m[key] = &pack
		pendingAcks.Unlock()

		return transmitVerbGenericUDP(node, nil, VERB_NFPING, code)
	}

	return nil
}

func receiveVerbPingUDP(msg message) error {
	return transmitVerbAckUDP(msg.sender, msg.senderCode)
}

func receiveVerbNonForwardPingUDP(msg message) error {
	return transmitVerbAckUDP(msg.sender, msg.senderCode)
}

func startTimeoutCheckLoop() {
	for {
		pendingAcks.Lock()
		for k, ack := range pendingAcks.m {
			elapsed := ack.Elapsed()

			if elapsed > TIMEOUT_MILLIS {
				fmt.Println(k, "timed out after", TIMEOUT_MILLIS, " milliseconds")

				// If a pending ack has a "downstream" field defined, then
				// it's the result of a NFP and we don't forward it. If it
				// isn't defined, we forward this request to a random host.

				if ack.Callback == nil {
					go doForwardOnTimeout(ack)
				} else {
					UpdateNodeStatus(ack.Callback, STATUS_DIED)
				}

				delete(pendingAcks.m, k)
			}
		}
		pendingAcks.Unlock()

		time.Sleep(time.Millisecond * 1000)
	}
}

func doForwardOnTimeout(pack *pendingAck) {
	random_nodes := liveNodes.getRandomNodes(forwardCount(), thisHost, pack.Node)

	if len(random_nodes) == 0 {
		fmt.Println(thisHost.Address(), "Cannot forward ping request: no more nodes")

		UpdateNodeStatus(pack.Node, STATUS_DIED)
	} else {
		for i, n := range random_nodes {
			fmt.Printf("(%d/%d) Requesting indirect ping of %s via %s\n",
				i+1,
				len(random_nodes),
				pack.Node.Address(),
				n.Address())

			transmitVerbForwardUDP(n, pack.Node, currentHeartbeat)
		}
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

	msg := newMessage(verb, thisHost, code)

	if forward_to != nil {
		msg.addMember(forward_to, STATUS_FORWARD_TO, code)
	}

	// Add members for update
	for _, m := range getRandomUpdatedNodes(forwardCount(), node) {
		msg.addMember(m, m.status, currentHeartbeat)
	}

	_, err = c.Write(msg.encode())
	if err != nil {
		return err
	}

	// Decrement the update counters on those nodes
	for _, m := range msg.members {
		m.node.broadcastCounter--
	}

	return nil
}

func transmitVerbForwardUDP(node *Node, downstream *Node, code uint32) error {
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)
	pack := pendingAck{Node: node, StartTime: GetNowInMillis(), Callback: downstream}

	pendingAcks.Lock()
	pendingAcks.m[key] = &pack
	pendingAcks.Unlock()

	return transmitVerbGenericUDP(node, downstream, VERB_FORWARD, code)
}

func transmitVerbAckUDP(node *Node, code uint32) error {
	return transmitVerbGenericUDP(node, nil, VERB_ACK, code)
}

func transmitVerbPingUDP(node *Node, code uint32) error {
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)
	pack := pendingAck{Node: node, StartTime: GetNowInMillis()}

	pendingAcks.Lock()
	pendingAcks.m[key] = &pack
	pendingAcks.Unlock()

	return transmitVerbGenericUDP(node, nil, VERB_PING, code)
}

func updateStatusesFromMessage(msg message) {
	for _, m := range msg.members {
		switch m.status {
		case STATUS_FORWARD_TO:
			// The FORWARD_TO status isn't useful here, so we ignore those
		case STATUS_DIED:
			// The DIED status doesn't add nodes to the liveNodes map
			UpdateNodeStatus(msg.sender, m.status)
		default:
			liveNodes.add(msg.sender)
			UpdateNodeStatus(msg.sender, m.status)
		}

		if m.status != STATUS_FORWARD_TO {
			if !liveNodes.contains(msg.sender) {
				liveNodes.add(msg.sender)
			}

			UpdateNodeStatus(msg.sender, m.status)
		}
	}

	// First, if we don't know the sender, we add it. Since it may have just
	// rejoined the cluster from a dead state, we report its status as ALIVE.
	if !liveNodes.contains(msg.sender) {
		liveNodes.add(msg.sender)
		UpdateNodeStatus(msg.sender, STATUS_ALIVE)
	}
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
