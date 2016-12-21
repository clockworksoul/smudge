package blackfish

import (
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
	if err != nil || ip == nil {
		logWarn("Warning: Could not resolve host IP")
	} else {
		me := Node{
			ip:         ip,
			port:       uint16(GetListenPort()),
			heartbeats: currentHeartbeat,
			timestamp:  GetNowInMillis(),
			status:     STATUS_ALIVE,
		}

		thisHostAddress = me.Address()
		thisHost = &me

		logInfo("My host address:", thisHostAddress)
		logInfo("My host:", thisHost)
	}

	// Add this node's status. Don't update any other node's statuses: they'll
	// report those back to us.
	AddNode(thisHost)
	UpdateNodeStatus(thisHost, STATUS_ALIVE)

	go listenUDP(GetListenPort())

	go startTimeoutCheckLoop()

	for {
		currentHeartbeat++

		logfDebug("[%d] %d hosts\n", currentHeartbeat, liveNodes.length())

		// Ping one random node
		nodes := getTargetNodes(1, thisHost)

		if len(nodes) >= 1 {
			PingNode(nodes[0])
		} else {
			logInfo("No nodes to ping. So lonely. :(")
		}

		// 1 heartbeat in 10, we resurrect a random dead node
		// if currentHeartbeat%25 == 0 {
		// 	ResurrectDeadNode()
		// }

		time.Sleep(time.Millisecond * time.Duration(GetHeartbeatMillis()))
	}
}

func PingAllNodes() {
	logDebug(liveNodes.length(), "nodes")

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
		logInfo("Failure to ping", node, "->", err)
	}

	return err
}

/******************************************************************************
 * Private functions (for internal use only)
 *****************************************************************************/

// Returns a random slice of valid ping/forward request targets; i.e., not
// this node, and not dead.
func getTargetNodes(count int, exclude ...*Node) []*Node {
	randomNodes := liveNodes.getRandomNodes(0, exclude...)
	filteredNodes := make([]*Node, 0, count)

	for _, n := range randomNodes {
		if len(filteredNodes) >= count {
			break
		}

		if n.status == STATUS_DIED {
			continue
		}

		filteredNodes = append(filteredNodes, n)
	}

	return filteredNodes
}

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
			logError("UDP read error: ", err)
		}

		go func(addr *net.UDPAddr, msg []byte) {
			err = receiveMessageUDP(addr, buf[0:n])
			if err != nil {
				logInfo(err)
			}
		}(addr, buf[0:n])
	}
}

func receiveMessageUDP(addr *net.UDPAddr, msg_bytes []byte) error {
	msg, err := decodeMessage(addr, msg_bytes)
	if err != nil {
		return err
	}

	logfTrace("Got %v from %v code=%d\n",
		msg.verb,
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

		msg.sender.heartbeats = currentHeartbeat
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
			Callback:     msg.sender,
			CallbackCode: code,
			packType:     PACK_NFP}

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
		for k, pack := range pendingAcks.m {
			elapsed := pack.Elapsed()

			if elapsed > TIMEOUT_MILLIS {
				// If a pending ack has a "downstream" field defined, then
				// it's the result of a NFP and we don't forward it. If it
				// isn't defined, we forward this request to a random host.

				switch pack.packType {
				case PACK_PING:
					go doForwardOnTimeout(pack)
				case PACK_FORWARD:
					logInfo(k, "timed out after", TIMEOUT_MILLIS, "milliseconds")
					UpdateNodeStatus(pack.Callback, STATUS_DIED)
				case PACK_NFP:
					logInfo(k, "timed out after", TIMEOUT_MILLIS, "milliseconds")
					UpdateNodeStatus(pack.Node, STATUS_DIED)
				}

				delete(pendingAcks.m, k)
			}
		}
		pendingAcks.Unlock()

		time.Sleep(time.Millisecond * 500)
	}
}

func doForwardOnTimeout(pack *pendingAck) {
	filteredNodes := getTargetNodes(forwardCount(), thisHost, pack.Node)

	if len(filteredNodes) == 0 {
		logInfo(thisHost.Address(), "Cannot forward ping request: no more nodes")

		UpdateNodeStatus(pack.Node, STATUS_DIED)
	} else {
		for i, n := range filteredNodes {
			logfInfo("(%d/%d) Requesting indirect ping of %s via %s\n",
				i+1,
				len(filteredNodes),
				pack.Node.Address(),
				n.Address())

			transmitVerbForwardUDP(n, pack.Node, currentHeartbeat)
		}
	}
}

func transmitVerbGenericUDP(node *Node, forward_to *Node, verb messageVerb, code uint32) error {
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

	logfTrace("Sent %v to %v\n", verb, node.Address())

	return nil
}

func transmitVerbForwardUDP(node *Node, downstream *Node, code uint32) error {
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)

	pack := pendingAck{
		Node:      node,
		StartTime: GetNowInMillis(),
		Callback:  downstream,
		packType:  PACK_FORWARD}

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
	pack := pendingAck{
		Node:      node,
		StartTime: GetNowInMillis(),
		packType:  PACK_PING}

	pendingAcks.Lock()
	pendingAcks.m[key] = &pack
	pendingAcks.Unlock()

	return transmitVerbGenericUDP(node, nil, VERB_PING, code)
}

func updateStatusesFromMessage(msg message) {
	for _, m := range msg.members {
		// logfDebug("Member update (%d/%d): %s is now status %s\n",
		// 	i+1,
		// 	len(msg.members),
		// 	m.node.Address(),
		// 	m.status)

		switch m.status {
		case STATUS_FORWARD_TO:
			// The FORWARD_TO status isn't useful here, so we ignore those
			continue
		case STATUS_DIED:
			// The DIED status doesn't add nodes to the liveNodes map
			UpdateNodeStatus(m.node, m.status)
		default:
			UpdateNodeStatus(m.node, m.status)
			AddNode(m.node)
		}
	}

	// First, if we don't know the sender, we add it. Since it may have just
	// rejoined the cluster from a dead state, we report its status as ALIVE.
	if !liveNodes.contains(msg.sender) {
		UpdateNodeStatus(msg.sender, STATUS_ALIVE)
		AddNode(msg.sender)
	}
}

type pendingAckType byte

const (
	PACK_PING pendingAckType = iota
	PACK_FORWARD
	PACK_NFP
)

func (p pendingAckType) String() string {
	switch p {
	case PACK_PING:
		return "PING"
	case PACK_FORWARD:
		return "FWD"
	case PACK_NFP:
		return "NFP"
	default:
		return "UNDEFINED"
	}
}

type pendingAck struct {
	StartTime    uint32
	Node         *Node
	Callback     *Node
	CallbackCode uint32
	packType     pendingAckType
}

func (a *pendingAck) Elapsed() uint32 {
	return GetNowInMillis() - a.StartTime
}
