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
var liveNodes nodeMap = nodeMap{}

// Recently dead nodes. Periodically a random dead node will be allowed to
// rejoin the living.
var deadNodes []*Node = make([]*Node, 0, 64)

var recently_updated []*Node = make([]*Node, 0, 64)

func init() {
	liveNodes.init()
}

/******************************************************************************
 * Exported functions (for public consumption)
 *****************************************************************************/

// Adds a node. Returns node, error.
// Updates node heartbeat in the process, but DOES NOT implicitly update the
// node's status; you need to do this explicitly.
func AddNode(node *Node) (*Node, error) {
	_, n, err := liveNodes.add(node)

	return n, err
}

// Given a node address ("ip:port" string), creates and returns a new node
// instance.
func CreateNodeByAddress(address string) (*Node, error) {
	ip, port, err := parseNodeAddress(address)

	if err == nil {
		node := Node{ip: ip, port: port, timestamp: GetNowInMillis()}

		return &node, nil
	}

	return nil, err
}

// Given a node address IP address and port, this function creates and returns
// a new node instance.
func CreateNodeByIP(ip net.IP, port uint16) (*Node, error) {
	return &Node{ip: ip, port: port, timestamp: GetNowInMillis()}, nil
}

// Queries the host interface to determine the local IPv4 of this machine.
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
func UpdateNodeStatus(n *Node, status NodeStatus) {
	if n.status != status {
		if status == STATUS_DIED {
			fmt.Printf("Node removed: %v\n", n)

			liveNodes.delete(n)

			deadNodes = append(deadNodes, n)
		}

		n.timestamp = GetNowInMillis()
		n.status = status
		n.broadcastCounter = byte(announceCount())

		fmt.Printf("[%s] status is now %v\n", n.Address(), n.status)

		// If this isn't in the recently updated list, add it.

		contains := false

		for _, ru := range recently_updated {
			if ru.Address() == n.Address() {
				contains = true
				break
			}
		}

		if !contains {
			recently_updated = append(recently_updated, n)
		}
	}

	fmt.Println("Updated", n.Address(), "to status", n.status)
}

/******************************************************************************
 * Private functions (for internal use only)
 *****************************************************************************/

// How many times a node should be broadcast after its status is updated.
func announceCount() int {
	var count int
	var lamda float64 = 2.5 // TODO Parameterize this magic number (lambda)

	if liveNodes.length() > 0 {
		logn := math.Log(float64(liveNodes.length()))

		mult := lamda * logn

		count = int(mult)
	}

	return count
}

// The number of nodes to request a forward of when a PING times out.
func forwardCount() int {
	var count int
	var lamda float64 = 2.5 // TODO Parameterize this magic number (lambda)

	if liveNodes.length() > 0 {
		logn := math.Log(float64(liveNodes.length()))

		mult := lamda * logn

		count = int(mult)
	}

	return count
}

func getRandomUpdatedNodes(size int, exclude ...*Node) []*Node {
	// First, prune those with broadcast counters of zero from the list

	pruned := make([]*Node, 0, len(recently_updated))
	for _, n := range recently_updated {
		if n.broadcastCounter > 0 {
			pruned = append(pruned, n)
		} else {
			fmt.Println("Removing", n.Address(), "from recently updated list")
		}
	}
	recently_updated = pruned

	// Make a copy of the recently update nodes slice

	updated_copy := make([]*Node, len(recently_updated), len(recently_updated))
	copy(updated_copy, recently_updated)

	// Exclude the exclusions
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

// Returns a random dead node to the liveNodes map.
func resurrectDeadNode() {
	if len(deadNodes) == 1 {
		liveNodes.add(deadNodes[0])
		deadNodes = make([]*Node, 0, 64)
	} else if len(deadNodes) > 1 {
		i := rand.Intn(len(deadNodes))
		liveNodes.add(deadNodes[i])

		dsub := deadNodes[:i]
		deadNodes := deadNodes[i+1:]

		for _, dn := range dsub {
			deadNodes = append(deadNodes, dn)
		}
	}
}
