package blackfish

import (
	"errors"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

// A scalar value used to calculate a variety of limits
const LAMBDA = 2.5

// Currenty active nodes
var liveNodes = nodeMap{}

// Recently dead nodes. Periodically a random dead node will be allowed to
// rejoin the living
var deadNodes = nodeMap{}

// All nodes that have been updated "recently", living and dead
var updatedNodes = nodeMap{}

func init() {
	liveNodes.init()
	deadNodes.init()
	updatedNodes.init()
}

/******************************************************************************
 * Exported functions (for public consumption)
 *****************************************************************************/

// AddNode can be used to explicity add a node to the list of known live nodes.
// Updates the node timestamp but DOES NOT implicitly update the node's status;
// you need to do this explicitly.
func AddNode(node *Node) (*Node, error) {
	if !liveNodes.contains(node) {
		logfInfo("Adding host: %s (status=%s)\n", node.Address(), node.status)

		if node.status == StatusUnknown {
			logWarn(node.Address(), "does not have a status!")
		} else if node.status == StatusForwardTo {
			panic("Invalid status: " + StatusForwardTo.String())
		}

		node.Touch()

		_, n, err := liveNodes.add(node)

		return n, err
	}

	return node, nil
}

// CreateNodeByAddress will create and return a new node when supplied with a
// node address ("ip:port" string). This doesn't add the node to the list of
// live nodes; use AddNode().
func CreateNodeByAddress(address string) (*Node, error) {
	ip, port, err := parseNodeAddress(address)

	if err == nil {
		node := Node{ip: ip, port: port, timestamp: GetNowInMillis()}

		return &node, nil
	}

	return nil, err
}

// CreateNodeByAddress will create and return a new node when supplied with an
// IP address and port number. This doesn't add the node to the list of live
// nodes; use AddNode().
func CreateNodeByIP(ip net.IP, port uint16) (*Node, error) {
	return &Node{ip: ip, port: port, timestamp: GetNowInMillis()}, nil
}

// GetLocalIP queries the host interface to determine the local IPv4 of this
// machine. If a local IPv4 cannot be found, then nil is returned. If the
// query to the underlying OS fails, an error is returned.
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

// RemoveNode can be used to explicity remove a node from the list of known
// live nodes. Updates the node timestamp but DOES NOT implicitly update the
// node's status; you need to do this explicitly.
func RemoveNode(node *Node) (*Node, error) {
	if !liveNodes.contains(node) {
		logfInfo("Removing host: %s (status=%s)\n", node.Address(), node.status)

		node.Touch()

		_, n, err := liveNodes.delete(node)

		return n, err
	}

	return node, nil
}

// UpdateNodeStatus assigns a new status for the specified node and adds it to
// the list of recently updated nodes. If the status is StatusDead, then the
// node will be moved from the live nodes list to the dead nodes list.
func UpdateNodeStatus(n *Node, status NodeStatus) {
	if n.status != status {
		n.timestamp = GetNowInMillis()
		n.status = status
		n.broadcastCounter = byte(announceCount())

		if n.status == StatusDead {
			logfInfo("Node removed: %s\n", n.Address())

			RemoveNode(n)
			deadNodes.add(n)
		}

		// If this isn't in the recently updated list, add it.
		if !updatedNodes.contains(n) {
			updatedNodes.add(n)
		}

		doStatusUpdate(n, status)
	}
}

/******************************************************************************
 * Private functions (for internal use only)
 *****************************************************************************/

func getRandomUpdatedNodes(size int, exclude ...*Node) []*Node {
	// Prune nodes with broadcast counters of 0 (or less) from the list
	for _, n := range updatedNodes.values() {
		if n.broadcastCounter <= 0 {
			logInfo("Removing", n.Address(), "from recently updated list")
			updatedNodes.delete(n)
		}
	}

	// Make a copy of the recently update nodes slice

	updatedCopy := updatedNodes.values()

	// Exclude the exclusions
	// TODO This is stupid inefficient. Use a set implementation of
	// some kind instead.

Outer:
	for _, nout := range exclude {
		for i, nin := range updatedCopy {
			if nout.Address() == nin.Address() {
				tmp := updatedCopy[0:i]

				for j := i + 1; j < len(updatedCopy); j++ {
					tmp = append(tmp, updatedCopy[j])
				}

				updatedCopy = tmp

				continue Outer
			}
		}
	}

	// Shuffle the copy

	for i := range updatedCopy {
		j := rand.Intn(i + 1)
		updatedCopy[i], updatedCopy[j] = updatedCopy[j], updatedCopy[i]
	}

	// Grab and return the top N

	if size > len(updatedCopy) {
		size = len(updatedCopy)
	}

	return updatedCopy[:size]
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

		if ip == nil {
			logWarn("Warning: Could not resolve host IP. Using 127.0.0.1")
			ip = []byte{127, 0, 0, 1}
		}
	}

	return ip, port, err
}

// Returns a random dead node to the liveNodes map.
func resurrectDeadNode() {
	randomDeadNode := deadNodes.getRandomNode()

	deadNodes.delete(randomDeadNode)
	AddNode(randomDeadNode)
}
