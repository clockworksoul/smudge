package blackfish

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

// Currenty active nodes.
var live_nodes NodeMap = NodeMap{}

// Recently dead nodes. Periodically a random dead node will be allowed to
// rejoin the living.
var dead_nodes []*Node = make([]*Node, 0, 64)

var recently_updated []*Node = make([]*Node, 0, 64)

func init() {
	live_nodes.init()
}

/******************************************************************************
 * Exported functions (for public consumption)
 *****************************************************************************/

// Adds a node. Returns node, error.
// Updates node heartbeat in the process, but DOES NOT implicitly update the
// node's status; you need to do this explicitly.
// This is the method called by all Add* functions.
//
func AddNode(node *Node) (*Node, error) {
	_, n, err := live_nodes.add(node)

	if node.Address() == this_host_address {
		fmt.Println("Rejecting node addition: it's this node")
	}

	return n, err
}

// Given a node address ("ip:port" string), creates and returns a new node
// instance.
//
func CreateNodeByAddress(address string) (*Node, error) {
	ip, port, err := parseNodeAddress(address)

	if err == nil {
		node := Node{IP: ip, Port: port, Timestamp: GetNowInMillis()}

		return &node, nil
	}

	return nil, err
}

// Given a node address IP address and port, this function creates and returns
// a new node instance.
//
func CreateNodeByIP(ip net.IP, port uint16) (*Node, error) {
	return &Node{IP: ip, Port: port, Timestamp: GetNowInMillis()}, nil
}

// Queries the host interface to determine the local IPv4 of this machine.
//
func GetLocalIP() (net.IP, error) {
	var ip net.IP

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ip, err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.To4()
				break
			}
		}
	}

	return ip, nil
}

// Assigns a new status for the specified node, and adds that node to the
// recently_updated list.
//
func UpdateNodeStatus(n *Node, status byte) {
	if n.status != status {
		fmt.Printf("[%s] status is now %s\n", n.Address(), n.StatusString())

		if status == STATUS_DIED {
			fmt.Printf("Node removed [%d > %d]: %v\n", n.Age(), GetDeadMillis(), n)

			live_nodes.delete(n)

			dead_nodes = append(dead_nodes, n)
		}

		n.Timestamp = GetNowInMillis()
		n.status = status
		n.broadcast_counter = byte(announceCount())
		recently_updated = append(recently_updated, n)
	}
}

/******************************************************************************
 * Private functions (for internal use only)
 *****************************************************************************/

// How many times a node should be broadcast after its status is updated.
func announceCount() int {
	var count int
	var lamda float64 = 2.5 // TODO Parameterize this magic number (lambda)

	if live_nodes.length() > 0 {
		logn := math.Log(float64(live_nodes.length()))

		mult := lamda * logn

		count = int(mult)
	}

	return count
}

// The number of nodes to request a forward of when a PING times out.
func forwardCount() int {
	var count int
	var lamda float64 = 2.5 // TODO Parameterize this magic number (lambda)

	if live_nodes.length() > 0 {
		logn := math.Log(float64(live_nodes.length()))

		mult := lamda * logn

		count = int(mult)
	}

	return count
}

func getRandomUpdatedNodes(size int, exclude ...*Node) []*Node {
	// Make a copy of the recently update nodes slice
	updated_copy := make([]*Node, len(recently_updated), len(recently_updated))
	copy(updated_copy, recently_updated)

	// Exclude the exclusions.
	// TODO This is stupid inefficient. Use a set implementation of
	// some kind instead.
Outer:
	for _, nout := range exclude {
		for i, nin := range updated_copy {
			if nout.Address() == nin.Address() {
				tmp := updated_copy[0:i]

				for j := i + 1; j < len(updated_copy); j++ {
					tmp = append(tmp, updated_copy[j])
				}

				updated_copy = tmp

				continue Outer
			}
		}
	}

	// Shuffle the copy
	for i := range updated_copy {
		j := rand.Intn(i + 1)
		updated_copy[i], updated_copy[j] = updated_copy[j], updated_copy[i]
	}

	// Grab and return the top N
	if size > len(updated_copy) {
		size = len(updated_copy)
	}

	return updated_copy[:size]
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
		port = uint16(GetListenPort())
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return ip, port, err
	}

	for _, i := range ips {
		if i.To4() != nil {
			ip = i.To4()
		}
	}

	if ip.IsLoopback() {
		ip, err = GetLocalIP()
	}

	return ip, port, err
}

// Returns a random dead node to the live_nodes map.
func resurrectDeadNode() {
	if len(dead_nodes) == 1 {
		live_nodes.add(dead_nodes[0])
		dead_nodes = make([]*Node, 0, 64)
	} else if len(dead_nodes) > 1 {
		i := rand.Intn(len(dead_nodes))
		live_nodes.add(dead_nodes[i])

		dsub := dead_nodes[:i]
		dead_nodes := dead_nodes[i+1:]

		for _, dn := range dsub {
			dead_nodes = append(dead_nodes, dn)
		}
	}
}

func updateStatusesFromMessage(msg message) {
	// First, if we don't know the sender, we add it.
	if !live_nodes.contains(msg.sender) {
		live_nodes.add(msg.sender)
	}

	UpdateNodeStatus(msg.sender, STATUS_ALIVE)

	for _, m := range msg.members {
		// The FORWARD_TO status isn't useful, so we actually ignore those
		if m.status != STATUS_FORWARD_TO {
			if !live_nodes.contains(msg.sender) {
				live_nodes.add(msg.sender)
			}

			UpdateNodeStatus(msg.sender, m.status)
		}
	}
}
