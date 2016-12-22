package blackfish

import (
	"errors"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

// All known nodes, living and dead. Dead nodes are pinged (far) less often,
// and are eventually removed
var knownNodes = nodeMap{}

// All nodes that have been updated "recently", living and dead
var updatedNodes = nodeMap{}

var deadNodeRetries = make(map[string]*deadNodeCounter)

const maxDeadNodeRetries = 10

func init() {
	knownNodes.init()
	updatedNodes.init()
}

/******************************************************************************
 * Exported functions (for public consumption)
 *****************************************************************************/

// AddNode can be used to explicitly add a node to the list of known live
// nodes. Updates the node timestamp but DOES NOT implicitly update the node's
// status; you need to do this explicitly.
func AddNode(node *Node) (*Node, error) {
	if !knownNodes.contains(node) {
		if node.status == StatusUnknown {
			logWarn(node.Address(),
				"does not have a status! Setting to",
				StatusAlive)

			UpdateNodeStatus(node, StatusAlive)
		} else if node.status == StatusForwardTo {
			panic("invalid status: " + StatusForwardTo.String())
		}

		node.Touch()

		_, n, err := knownNodes.add(node)

		logfInfo("Adding host: %s (total=%d live=%d dead=%d)\n",
			node.Address(),
			knownNodes.length(),
			knownNodes.lengthWithStatus(StatusAlive),
			knownNodes.lengthWithStatus(StatusDead))

		knownNodesModifiedFlag = true

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

// CreateNodeByIP will create and return a new node when supplied with an
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

// RemoveNode can be used to explicitly remove a node from the list of known
// live nodes. Updates the node timestamp but DOES NOT implicitly update the
// node's status; you need to do this explicitly.
func RemoveNode(node *Node) (*Node, error) {
	if knownNodes.contains(node) {
		node.Touch()

		_, n, err := knownNodes.delete(node)

		logfInfo("Removing host: %s (total=%d live=%d dead=%d)\n",
			node.Address(),
			knownNodes.length(),
			knownNodes.lengthWithStatus(StatusAlive),
			knownNodes.lengthWithStatus(StatusDead))

		knownNodesModifiedFlag = true

		return n, err
	}

	return node, nil
}

// UpdateNodeStatus assigns a new status for the specified node and adds it to
// the list of recently updated nodes. If the status is StatusDead, then the
// node will be moved from the live nodes list to the dead nodes list.
func UpdateNodeStatus(node *Node, status NodeStatus) {
	if node.status != status {
		node.timestamp = GetNowInMillis()
		node.status = status
		node.broadcastCounter = byte(announceCount())

		// If this isn't in the recently updated list, add it.
		if !updatedNodes.contains(node) {
			updatedNodes.add(node)
		}

		if status != StatusDead {
			delete(deadNodeRetries, node.Address())
		}

		logfInfo("Updating host: %s to %s (total=%d live=%d dead=%d)\n",
			node.Address(),
			status,
			knownNodes.length(),
			knownNodes.lengthWithStatus(StatusAlive),
			knownNodes.lengthWithStatus(StatusDead))

		doStatusUpdate(node, status)
	}
}

/******************************************************************************
 * Private functions (for internal use only)
 *****************************************************************************/

func getRandomUpdatedNodes(size int, exclude ...*Node) []*Node {
	// Prune nodes with broadcast counters of 0 (or less) from the list
	for _, n := range updatedNodes.values() {
		if n.broadcastCounter <= 0 {
			logDebug("Removing", n.Address(), "from recently updated list")
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

type deadNodeCounter struct {
	retry          int
	retryCountdown int
}
