/*
Copyright 2016 The Smudge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package smudge

import (
	"errors"
	"math"
	"net"
	"strconv"
	"sync"
	"time"
)

// A scalar value used to calculate a variety of limits
const lambda = 2.5

// How many standard deviations beyond the mean PING/ACK response time we
// allow before timing out an ACK.
const timeoutToleranceSigmas = 3.0

const defaultIPv4MulticastAddress = "224.0.0.0"

const defaultIPv6MulticastAddress = "[ff02::1]"

var currentHeartbeat uint32

var pendingAcks = struct {
	sync.RWMutex
	m map[string]*pendingAck
}{m: make(map[string]*pendingAck)}

var thisHostAddress string

var thisHost *Node

var ipLen = net.IPv4len

// This flag is set whenever a known node is added or removed.
var knownNodesModifiedFlag = false

var pingdata = newPingData(GetPingHistoryFrontload(), 50)

/******************************************************************************
 * Exported functions (for public consumption)
 *****************************************************************************/

// Begin starts the server by opening a UDP port and beginning the heartbeat.
// Note that this is a blocking function, so act appropriately.
func Begin() {
	// Add this host.
	logfInfo("Using listen IP: %s", listenIP)

	// Use IPv6 address length if the listen IP is not an IPv4 address
	if GetListenIP().To4() == nil {
		ipLen = net.IPv6len
	}

	me := Node{
		ip:         GetListenIP(),
		port:       uint16(GetListenPort()),
		timestamp:  GetNowInMillis(),
		pingMillis: PingNoData,
	}

	thisHostAddress = me.Address()
	thisHost = &me

	logInfo("My host address:", thisHostAddress)

	// Add this node's status. Don't update any other node's statuses: they'll
	// report those back to us.
	updateNodeStatus(thisHost, StatusAlive, 0, thisHost)
	AddNode(thisHost)

	go listenUDP(GetListenPort())

	// Add initial hosts as specified by the SMUDGE_INITIAL_HOSTS property
	for _, address := range GetInitialHosts() {
		n, err := CreateNodeByAddress(address)
		if err != nil {
			logfError("Could not create node %s: %v", address, err)
		} else {
			AddNode(n)
		}
	}

	if GetMulticastEnabled() {
		go listenUDPMulticast(GetMulticastPort())
		go multicastAnnounce(GetMulticastAddress())
	}

	go startTimeoutCheckLoop()

	// Loop over a randomized list of all known nodes (except for this host
	// node), pinging one at a time. If the knownNodesModifiedFlag is set to
	// true by AddNode() or RemoveNode(), the we get a fresh list and start
	// again.

	for {
		var randomAllNodes = knownNodes.getRandomNodes(0, thisHost)
		var pingCounter int

		for _, node := range randomAllNodes {
			// Exponential backoff of dead nodes, until such time as they are removed.
			if node.status == StatusDead {
				var dnc *deadNodeCounter
				var ok bool

				deadNodeRetries.Lock()
				if dnc, ok = deadNodeRetries.m[node.Address()]; !ok {
					dnc = &deadNodeCounter{retry: 1, retryCountdown: 2}
					deadNodeRetries.m[node.Address()] = dnc
				}
				deadNodeRetries.Unlock()

				dnc.retryCountdown--

				if dnc.retryCountdown <= 0 {
					dnc.retry++
					dnc.retryCountdown = int(math.Pow(2.0, float64(dnc.retry)))

					if dnc.retry > maxDeadNodeRetries {
						logDebug("Forgetting dead node", node.Address())

						deadNodeRetries.Lock()
						delete(deadNodeRetries.m, node.Address())
						deadNodeRetries.Unlock()

						RemoveNode(node)
						continue
					}
				} else {
					continue
				}
			}

			currentHeartbeat++

			logfTrace("%d - hosts=%d (announce=%d forward=%d)",
				currentHeartbeat,
				len(randomAllNodes),
				emitCount(),
				pingRequestCount())

			PingNode(node)
			pingCounter++

			time.Sleep(time.Millisecond * time.Duration(GetHeartbeatMillis()))

			if knownNodesModifiedFlag {
				knownNodesModifiedFlag = false
				break
			}
		}

		if pingCounter == 0 {
			logDebug("No nodes to ping. So lonely. :(")
			time.Sleep(time.Millisecond * time.Duration(GetHeartbeatMillis()))
		}
	}
}

// PingNode can be used to explicitly ping a node. Calls the low-level
// doPingNode(), and outputs a message (and returns an error) if it fails.
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

// Multicast announcements are constructed as:
// Byte  0      - 1 byte character byte length N
// Bytes 1 to N - Cluster name bytes
// Bytes N+1... - A message (without members)
func decodeMulticastAnnounceBytes(bytes []byte) (string, []byte, error) {
	nameBytesLen := int(bytes[0])

	if nameBytesLen+1 > len(bytes) {
		return "", nil, errors.New("Invalid multicast message received")
	}

	nameBytes := bytes[1 : nameBytesLen+1]
	name := string(nameBytes)
	msgBytes := bytes[nameBytesLen+1 : len(bytes)]

	return name, msgBytes, nil
}

func doForwardOnTimeout(pack *pendingAck) {
	filteredNodes := getTargetNodes(pingRequestCount(), thisHost, pack.node)

	if len(filteredNodes) == 0 {
		logDebug(thisHost.Address(), "Cannot forward ping request: no more nodes")

		updateNodeStatus(pack.node, StatusDead, currentHeartbeat, thisHost)
	} else {
		for i, n := range filteredNodes {
			logfDebug("(%d/%d) Requesting indirect ping of %s via %s",
				i+1,
				len(filteredNodes),
				pack.node.Address(),
				n.Address())

			transmitVerbForwardUDP(n, pack.node, currentHeartbeat)
		}
	}
}

// The number of times any node's new status should be emitted after changes.
// Currently set to (lambda * log(node count)).
func emitCount() int {
	logn := math.Log(float64(knownNodes.length()))
	mult := (lambda * logn) + 0.5

	return int(mult)
}

// Multicast announcements are constructed as:
// Byte  0      - 1 byte character byte length N
// Bytes 1 to N - Cluster name bytes
// Bytes N+1... - A message (without members)
func encodeMulticastAnnounceBytes() []byte {
	nameBytes := []byte(GetClusterName())
	nameBytesLen := len(nameBytes)

	if nameBytesLen > 0xFF {
		panic("Cluster name too long: " +
			strconv.FormatInt(int64(nameBytesLen), 10) +
			" bytes (max 254)")
	}

	msg := newMessage(verbPing, thisHost, currentHeartbeat)
	msgBytes := msg.encode()
	msgBytesLen := len(msgBytes)

	totalByteCount := 1 + nameBytesLen + msgBytesLen

	bytes := make([]byte, totalByteCount, totalByteCount)

	// Add name length byte
	bytes[0] = byte(nameBytesLen)

	// Copy the name bytes
	copy(bytes[1:nameBytesLen+1], nameBytes)

	// Copy the message proper
	copy(bytes[nameBytesLen+1:totalByteCount], msgBytes)

	return bytes
}

func guessMulticastAddress() string {
	if multicastAddress == "" {
		if ipLen == net.IPv6len {
			multicastAddress = defaultIPv6MulticastAddress
		} else if ipLen == net.IPv4len {
			multicastAddress = defaultIPv4MulticastAddress
		} else {
			logFatal("Failed to determine IPv4/IPv6")
		}
	}

	return multicastAddress
}

// Returns a random slice of valid ping/forward request targets; i.e., not
// this node, and not dead.
func getTargetNodes(count int, exclude ...*Node) []*Node {
	randomNodes := knownNodes.getRandomNodes(0, exclude...)
	filteredNodes := make([]*Node, 0, count)

	for _, n := range randomNodes {
		if len(filteredNodes) >= count {
			break
		}

		if n.status == StatusDead {
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
		buf := make([]byte, 2048) // big enough to fit 1280 IPv6 UDP message
		n, addr, err := c.ReadFromUDP(buf)
		if err != nil {
			logError("UDP read error: ", err)
		}

		go func(addr *net.UDPAddr, msg []byte) {
			err = receiveMessageUDP(addr, buf[0:n])
			if err != nil {
				logError(err)
			}
		}(addr, buf[0:n])
	}
}

func listenUDPMulticast(port int) error {
	addr := GetMulticastAddress()
	if addr == "" {
		addr = guessMulticastAddress()
	}

	listenAddress, err := net.ResolveUDPAddr("udp", addr+":"+strconv.FormatInt(int64(port), 10))
	if err != nil {
		return err
	}

	/* Now listen at selected port */
	c, err := net.ListenMulticastUDP("udp", nil, listenAddress)
	if err != nil {
		return err
	}
	defer c.Close()

	for {
		buf := make([]byte, 2048) // big enough to fit 1280 IPv6 UDP message
		n, addr, err := c.ReadFromUDP(buf)
		if err != nil {
			logError("UDP read error:", err)
		}

		go func(addr *net.UDPAddr, bytes []byte) {
			name, msgBytes, err := decodeMulticastAnnounceBytes(bytes)

			if err != nil {
				logDebug("Ignoring unexpected multicast message.")
			} else {
				if GetClusterName() == name {
					msg, err := decodeMessage(addr.IP, msgBytes)
					if err == nil {
						logfTrace("Got multicast %v from %v code=%d",
							msg.verb,
							msg.sender.Address(),
							msg.senderHeartbeat)

						// Update statuses of the sender.
						updateStatusesFromMessage(msg)
					} else {
						logError(err)
					}
				}
			}
		}(addr, buf[0:n])
	}
}

// multicastAnnounce is called when the server first starts to broadcast its
// presence to all listening servers within the specified subnet and continues
// to broadcast its presence every multicastAnnounceIntervalSeconds in case
// this value is larger than zero.
func multicastAnnounce(addr string) error {
	if addr == "" {
		addr = guessMulticastAddress()
	}

	fullAddr := addr + ":" + strconv.FormatInt(int64(GetMulticastPort()), 10)

	logInfo("Announcing presence on", fullAddr)

	address, err := net.ResolveUDPAddr("udp", fullAddr)
	if err != nil {
		logError(err)
		return err
	}

	for {
		c, err := net.DialUDP("udp", nil, address)
		if err != nil {
			logError(err)
			return err
		}

		// Compose and send the multicast announcement
		msgBytes := encodeMulticastAnnounceBytes()
		_, err = c.Write(msgBytes)
		if err != nil {
			logError(err)
			return err
		}

		logfTrace("Sent announcement multicast to %v", fullAddr)

		if GetMulticastAnnounceIntervalSeconds() > 0 {
			time.Sleep(time.Second * time.Duration(GetMulticastAnnounceIntervalSeconds()))
		} else {
			return nil
		}
	}
}

// The number of nodes to send a PINGREQ to when a PING times out.
// Currently set to (lambda * log(node count)).
func pingRequestCount() int {
	logn := math.Log(float64(knownNodes.length()))
	mult := (lambda * logn) + 0.5

	return int(mult)
}

func receiveMessageUDP(addr *net.UDPAddr, msgBytes []byte) error {
	msg, err := decodeMessage(addr.IP, msgBytes)
	if err != nil {
		return err
	}

	logfTrace("Got %v from %v code=%d",
		msg.verb,
		msg.sender.Address(),
		msg.senderHeartbeat)

	// Synchronize heartbeats
	if msg.senderHeartbeat > 0 && msg.senderHeartbeat-1 > currentHeartbeat {
		logfTrace("Heartbeat advanced from %d to %d",
			currentHeartbeat,
			msg.senderHeartbeat-1)

		currentHeartbeat = msg.senderHeartbeat - 1
	}

	// Update statuses of the sender and any members the message includes.
	updateStatusesFromMessage(msg)

	// If there are broadcast bytes in the message, handle them here.
	receiveBroadcast(msg.broadcast)

	// Handle the verb.
	switch msg.verb {
	case verbPing:
		err = receiveVerbPingUDP(msg)
	case verbAck:
		err = receiveVerbAckUDP(msg)
	case verbPingRequest:
		err = receiveVerbForwardUDP(msg)
	case verbNonForwardingPing:
		err = receiveVerbNonForwardPingUDP(msg)
	}

	if err != nil {
		return err
	}

	return nil
}

func receiveVerbAckUDP(msg message) error {
	key := msg.sender.Address() + ":" + strconv.FormatInt(int64(msg.senderHeartbeat), 10)

	pendingAcks.RLock()
	_, ok := pendingAcks.m[key]
	pendingAcks.RUnlock()

	if ok {
		msg.sender.Touch()

		pendingAcks.Lock()

		if pack, ok := pendingAcks.m[key]; ok {
			// If this is a response to a requested ping, respond to the
			// callback node
			if pack.callback != nil {
				go transmitVerbAckUDP(pack.callback, pack.callbackCode)
			} else {
				// Note the ping response time.
				notePingResponseTime(pack)
			}
		}

		delete(pendingAcks.m, key)
		pendingAcks.Unlock()
	}

	return nil
}

func notePingResponseTime(pack *pendingAck) {
	// Note the elapsed time
	elapsedMillis := pack.elapsed()

	pack.node.pingMillis = int(elapsedMillis)

	// For the purposes of timeout tolerance, we treat all pings less than
	// the ping lower bound as that lower bound.
	minMillis := uint32(GetMinPingTime())
	if elapsedMillis < minMillis {
		elapsedMillis = minMillis
	}

	pingdata.add(elapsedMillis)

	mean, stddev := pingdata.data()
	sigmas := pingdata.nSigma(timeoutToleranceSigmas)

	logfTrace("Got ACK in %dms (mean=%.02f stddev=%.02f sigmas=%.02f)",
		elapsedMillis,
		mean,
		stddev,
		sigmas)
}

func receiveVerbForwardUDP(msg message) error {
	// We don't forward to a node that we don't know.

	if len(msg.members) >= 0 &&
		msg.members[0].status == StatusForwardTo {

		member := msg.members[0]
		node := member.node
		code := member.heartbeat
		key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)

		pack := pendingAck{
			node:         node,
			startTime:    GetNowInMillis(),
			callback:     msg.sender,
			callbackCode: code,
			packType:     packNFP}

		pendingAcks.Lock()
		pendingAcks.m[key] = &pack
		pendingAcks.Unlock()

		return transmitVerbGenericUDP(node, nil, verbNonForwardingPing, code)
	}

	return nil
}

func receiveVerbPingUDP(msg message) error {
	return transmitVerbAckUDP(msg.sender, msg.senderHeartbeat)
}

func receiveVerbNonForwardPingUDP(msg message) error {
	return transmitVerbAckUDP(msg.sender, msg.senderHeartbeat)
}

func startTimeoutCheckLoop() {
	for {
		pendingAcks.Lock()
		for k, pack := range pendingAcks.m {
			elapsed := pack.elapsed()
			timeoutMillis := uint32(pingdata.nSigma(timeoutToleranceSigmas))

			// Ping requests are expected to take quite a bit longer.
			// Just call it 2x for now.
			if pack.packType == packPingReq {
				timeoutMillis *= 2
			}

			// This pending ACK has taken longer than expected. Mark it as
			// timed out.
			if elapsed > timeoutMillis {
				switch pack.packType {
				case packPing:
					go doForwardOnTimeout(pack)
				case packPingReq:
					logDebug(k, "timed out after", timeoutMillis, "milliseconds (dropped PINGREQ)")

					if knownNodes.contains(pack.callback) {
						switch pack.callback.Status() {
						case StatusDead:
							break
						case StatusSuspected:
							updateNodeStatus(pack.callback, StatusDead, currentHeartbeat, thisHost)
							pack.callback.pingMillis = PingTimedOut
						default:
							updateNodeStatus(pack.callback, StatusSuspected, currentHeartbeat, thisHost)
							pack.callback.pingMillis = PingTimedOut
						}
					}
				case packNFP:
					logDebug(k, "timed out after", timeoutMillis, "milliseconds (dropped NFP)")

					if knownNodes.contains(pack.node) {
						switch pack.node.Status() {
						case StatusDead:
							break
						case StatusSuspected:
							updateNodeStatus(pack.node, StatusDead, currentHeartbeat, thisHost)
							pack.callback.pingMillis = PingTimedOut
						default:
							updateNodeStatus(pack.node, StatusSuspected, currentHeartbeat, thisHost)
							pack.callback.pingMillis = PingTimedOut
						}
					}
				}

				delete(pendingAcks.m, k)
			}
		}
		pendingAcks.Unlock()

		time.Sleep(time.Millisecond * 100)
	}
}

func transmitVerbGenericUDP(node *Node, forwardTo *Node, verb messageVerb, code uint32) error {
	// Transmit the ACK
	remoteAddr, err := net.ResolveUDPAddr("udp", node.Address())

	c, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		return err
	}
	defer c.Close()

	msg := newMessage(verb, thisHost, code)

	if forwardTo != nil {
		msg.addMember(forwardTo, StatusForwardTo, code, forwardTo.statusSource)
	}

	// Add members for update.
	nodes := getRandomUpdatedNodes(pingRequestCount(), node, thisHost)

	// No updates to distribute? Send out a few updates on other known nodes.
	if len(nodes) == 0 {
		nodes = knownNodes.getRandomNodes(pingRequestCount(), node, thisHost)
	}

	for _, n := range nodes {
		err = msg.addMember(n, n.status, n.heartbeat, n.statusSource)
		if err != nil {
			return err
		}

		n.emitCounter--
	}

	// Emit counters for broadcasts can be less than 0. We transmit positive
	// numbers, and decrement all the others. At some value < 0, the broadcast
	// is removed from the map all together.
	broadcast := getBroadcastToEmit()
	if broadcast != nil {
		if broadcast.emitCounter > 0 {
			msg.addBroadcast(broadcast)
		}

		broadcast.emitCounter--
	}

	_, err = c.Write(msg.encode())
	if err != nil {
		return err
	}

	// Decrement the update counters on those nodes
	for _, m := range msg.members {
		m.node.emitCounter--
	}

	logfTrace("Sent %v to %v", verb, node.Address())

	return nil
}

func transmitVerbForwardUDP(node *Node, downstream *Node, code uint32) error {
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)

	pack := pendingAck{
		node:      node,
		startTime: GetNowInMillis(),
		callback:  downstream,
		packType:  packPingReq}

	pendingAcks.Lock()
	pendingAcks.m[key] = &pack
	pendingAcks.Unlock()

	return transmitVerbGenericUDP(node, downstream, verbPingRequest, code)
}

func transmitVerbAckUDP(node *Node, code uint32) error {
	return transmitVerbGenericUDP(node, nil, verbAck, code)
}

func transmitVerbPingUDP(node *Node, code uint32) error {
	key := node.Address() + ":" + strconv.FormatInt(int64(code), 10)
	pack := pendingAck{
		node:      node,
		startTime: GetNowInMillis(),
		packType:  packPing}

	pendingAcks.Lock()
	pendingAcks.m[key] = &pack
	pendingAcks.Unlock()

	return transmitVerbGenericUDP(node, nil, verbPing, code)
}

func updateStatusesFromMessage(msg message) {
	for _, m := range msg.members {
		// If the heartbeat in the message is less then the heartbeat
		// associated with the last known status, then we conclude that the
		// message is old and we drop it.
		if m.heartbeat < m.node.heartbeat {
			logfDebug("Message is old (%d vs %d): dropping",
				m.node.heartbeat, m.heartbeat)

			continue
		}

		switch m.status {
		case StatusForwardTo:
			// The FORWARD_TO status isn't useful here, so we ignore those.
			continue
		case StatusDead:
			// Don't tell ME I'm dead.
			if m.node.Address() != thisHost.Address() {
				updateNodeStatus(m.node, m.status, m.heartbeat, m.source)
				AddNode(m.node)
			}
		default:
			updateNodeStatus(m.node, m.status, m.heartbeat, m.source)
			AddNode(m.node)
		}
	}

	// Obviously, we know the sender is alive. Report it as such.
	if msg.senderHeartbeat > msg.sender.heartbeat {
		updateNodeStatus(msg.sender, StatusAlive, msg.senderHeartbeat, thisHost)
	}

	// Finally, if we don't know the sender we add it to the known hosts map.
	if !knownNodes.contains(msg.sender) {
		AddNode(msg.sender)
	}
}

// pendingAckType represents an expectation of a response to a previously
// emitted PING, PINGREQ, or NFP.
type pendingAck struct {
	startTime    uint32
	node         *Node
	callback     *Node
	callbackCode uint32
	packType     pendingAckType
}

func (a *pendingAck) elapsed() uint32 {
	return GetNowInMillis() - a.startTime
}

// pendingAckType represents the type of PING that a pendingAckType is waiting
// for a response for: PING, PINGREQ, or NFP.
type pendingAckType byte

const (
	packPing pendingAckType = iota
	packPingReq
	packNFP
)

func (p pendingAckType) String() string {
	switch p {
	case packPing:
		return "PING"
	case packPingReq:
		return "PINGREQ"
	case packNFP:
		return "NFP"
	default:
		return "UNDEFINED"
	}
}
