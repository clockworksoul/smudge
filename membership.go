package blackfish

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Currenty active nodes.
var live_nodes map[string]*Node

// Recently dead nodes. Periodically a random dead node will be allowed to
// rejoin the living.
var dead_nodes []Node

var current_heartbeat uint32

func GetNowInMillis() uint32 {
	return uint32(time.Now().UnixNano() / int64(time.Millisecond))
}

func generateIdentifier() string {
	bytes := make([]byte, 0, 1000)

	// Begin with the byte value of the current nano time
	//
	now_bytes, _ := time.Now().MarshalBinary()
	for _, b := range now_bytes {
		bytes = append(bytes, b)
	}

	// Append the hostname of the current machine
	//
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("WARNING: Could not resolve hostname. Using 'localhost'")
		hostname = "localhost"
	}

	hostname_bytes := make([]byte, len(hostname), len(hostname))
	copy(hostname_bytes[:], hostname)
	for _, b := range hostname_bytes {
		bytes = append(bytes, b)
	}

	// Append some random data
	//
	rand := rand.Int63()
	var i uint
	for i = 0; i < 8; i++ {
		bytes = append(bytes, byte(rand>>i))
	}

	sha256 := sha256.Sum256(bytes)

	return base64.StdEncoding.EncodeToString(sha256[:])
}

// Explicitly adds a node to this server's internal nodes list.
func AddNode(name string) {
	host, port, err := parseNodeAddress(name)

	if err != nil {
		fmt.Println("Failure to parse node name:", err)
		return
	}

	node := Node{Host: host, Port: port, Timestamp: GetNowInMillis()}

	registerNewNode(node)
}

func Begin() {
	go Listen(GetListenPort())

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

func ResurrectDeadNode() {
	if len(dead_nodes) == 1 {
		registerNewNode(dead_nodes[0])
		dead_nodes = make([]Node, 0, 64)
	} else if len(dead_nodes) > 1 {
		i := rand.Intn(len(dead_nodes))
		registerNewNode(dead_nodes[i])

		dsub := dead_nodes[:i]
		dead_nodes := dead_nodes[i+1:]

		for _, dn := range dsub {
			dead_nodes = append(dead_nodes, dn)
		}
	}
}

// Loops through the nodes map and removes the dead ones.
func PruneDeadFromList() {
	for k, n := range live_nodes {
		if n.Age() > uint32(GetDeadMillis()) {
			fmt.Printf("Node removed [%d > %d]: %v\n", n.Age(), GetDeadMillis(), n)

			delete(live_nodes, k)
			dead_nodes = append(dead_nodes, *n)
		}
	}
}

// Starts the server on the indicated node. This is a blocking operation,
// so you probably want to execute this as a gofunc.
func Listen(port int) error {
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
	err := doPingNode(node)
	if err != nil {
		fmt.Println("Failure to ping", node, "->", err)
	}

	return err
}

func doPingNode(node *Node) error {
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

// Returns a slice of Node[] of from 0 to len(nodes) nodes.
// If size is < len(nodes), that many nodes are randomly chosen and
// returned.
func GetRandomNodes(size int) *[]Node {
	emptySlice := make([]Node, 0, 0)

	return getRandomNodes(size, &emptySlice)
}

// Returns a slice of Node[] of from 0 to len(nodes) nodes.
// If size is < len(nodes), that many nodes are randomly chosen and
// returned.
func getRandomNodes(size int, exclude *[]Node) *[]Node {
	excludeMap := make(map[string]*Node)
	for _, n := range *exclude {
		excludeMap[n.Address()] = &n
	}

	// If size is less than the entire set of nodes, shuffle and get a subset.
	if size <= 0 || size > len(live_nodes) {
		size = len(live_nodes)
	}

	// Copy the complete nodes map into a slice
	rnodes := make([]Node, 0, size)

	var c int
	for _, n := range live_nodes {
		// If a node is not stale...
		//
		if n.Age() < uint32(GetStaleMillis()) {
			// And isn't in the excludes map...
			//
			// if _, ok := nodes[n.Address()]; !ok {
			// We append it
			rnodes = append(rnodes, *n)
			c++

			if c >= size {
				break
				// }
			}
		}
	}

	if size < len(rnodes) {
		// Shuffle the slice
		for i := range rnodes {
			j := rand.Intn(i + 1)
			rnodes[i], rnodes[j] = rnodes[j], rnodes[i]
		}

		rnodes = rnodes[0:size]
	}

	return &rnodes
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

// Merges a slice of nodes into the nodes map.
// Returns a slice of the nodes that were merged or updated (or ignored for
// having exactly equal heartbeats)
func mergeNodeLists(msgNodes *[]Node) *[]Node {
	mergedNodes := make([]Node, 0, 1)

	for _, msgNode := range *msgNodes {
		if existingNode, ok := live_nodes[msgNode.Address()]; ok {
			// If the heartbeats are exactly equal, we don't merge the node,
			// but since we also don't want to retarnsmit it back to its source
			// we add to the slice of 'merged' nodes.
			if msgNode.Heartbeats >= existingNode.Heartbeats {
				mergedNodes = append(mergedNodes, *existingNode)
			}

			if msgNode.Heartbeats > existingNode.Heartbeats {
				// We have this node in our list. Touch it to update the timestamp.
				//
				existingNode.Heartbeats = msgNode.Heartbeats
				existingNode.Touch()

				fmt.Printf("[%s] Node exists; is now: %v\n",
					msgNode.Address(), existingNode)
			} else {
				fmt.Printf("[%s] Node exists but heartbeat is older; ignoring\n",
					msgNode.Address())
			}
		} else {
			// We do not have this node in our list. Add it.
			fmt.Println("New node identified:", msgNode)
			registerNewNode(msgNode)

			if msgNode.Heartbeats >= existingNode.Heartbeats {
				mergedNodes = append(mergedNodes, *existingNode)
			}
		}
	}

	return &mergedNodes
}

func receiveNodes(decoder *gob.Decoder) (*[]Node, error) {
	var mnodes []Node

	// Second, receive the list
	//
	var length int
	var host string
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

		ip, _, err := parseNodeAddress(host) 
		if err != nil {
			fmt.Printf("Error parsing host address (%s): %v\n", host, err)
			return &mnodes, err
		}

		newNode := Node{
			Host:       ip,
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

	// Finally, merge the listy of nodes we received from the peer into ours
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

func init() {
	if live_nodes == nil {
		live_nodes = make(map[string]*Node)
		dead_nodes = make([]Node, 0, 64)
	}
}

func parseNodeAddress(hostAndMaybePort string) (net.IP, uint16, error) {
	var host string
	var ip net.IP
	var port uint16
	var err error

	if strings.Contains(hostAndMaybePort, ":") {
		splode := strings.Split(hostAndMaybePort, ":")

		if len(splode) == 2 {
			p, e := strconv.ParseUint(splode[1], 10, 16)

			host = splode[0]
			port = uint16(p)
			err = e
		} else {
			err = errors.New("too many colons in argument " + hostAndMaybePort)
		}
	} else {
		host = hostAndMaybePort
		port = uint16(DEFAULT_LISTEN_PORT)
	}

	ips, err := net.LookupIP(host)
	if err != nil  {
		return ip, port, err
	}

	for _, i := range ips {
		if i.To4() != nil {
			ip = i
		}
	}

	return ip, port, err
}

func registerNewNode(node Node) {
	fmt.Println("Adding host:", node.Address())

	node.Touch()
	node.Heartbeats = current_heartbeat

	live_nodes[node.Address()] = &node

	for k, v := range live_nodes {
		fmt.Printf(" k=%s v=%v\n", k, v)
	}
}
