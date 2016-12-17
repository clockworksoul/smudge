package blackfish

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
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

func AddNodeByAddress(address string) {
	live_nodes.AddByAddress(address)
}

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

// Returns a random dead node to the live_nodes map.
func ResurrectDeadNode() {
	if len(dead_nodes) == 1 {
		live_nodes.Add(dead_nodes[0])
		dead_nodes = make([]*Node, 0, 64)
	} else if len(dead_nodes) > 1 {
		i := rand.Intn(len(dead_nodes))
		live_nodes.Add(dead_nodes[i])

		dsub := dead_nodes[:i]
		dead_nodes := dead_nodes[i+1:]

		for _, dn := range dsub {
			dead_nodes = append(dead_nodes, dn)
		}
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

// How many times a node should be broadcast after its status is updated.
func announceCount() int {
	var count int

	if live_nodes.Length() > 0 {
		logn := math.Log(float64(live_nodes.Length()))
		mult := 2.5 * logn

		count = int(mult)
	}

	return count
}

// The number of nodes to request a forward of when a PING times out.
func forwardCount() int {
	var count int

	if live_nodes.Length() > 0 {
		logn := math.Log(float64(live_nodes.Length()))
		mult := 2.5 * logn

		count = int(mult)
	}

	return count
}

func UpdateNodeStatus(n *Node, status byte) {
	if n.status != status {
		fmt.Printf("[%s] status is now %s\n", n.Address(), n.StatusString())

		if status == STATUS_DIED {
			fmt.Printf("Node removed [%d > %d]: %v\n", n.Age(), GetDeadMillis(), n)

			live_nodes.Delete(n)

			dead_nodes = append(dead_nodes, n)
		}

		n.Timestamp = GetNowInMillis()
		n.status = status
		n.broadcast_counter = byte(announceCount())
		recently_updated = append(recently_updated, n)
	}
}

func updateStatusesFromMessage(msg message) {
	// First, if we don't know the sender, we add it.
	if !live_nodes.Contains(msg.sender) {
		live_nodes.Add(msg.sender)
	}

	UpdateNodeStatus(msg.sender, STATUS_ALIVE)

	for _, m := range msg.members {
		// The FORWARD_TO status isn't useful, so we actually ignore those
		if m.status != STATUS_FORWARD_TO {
			if !live_nodes.Contains(msg.sender) {
				live_nodes.Add(msg.sender)
			}

			UpdateNodeStatus(msg.sender, m.status)
		}
	}
}

func getRandomUpdatedNodes(size int) []*Node {
	// Make a copy of the recently update nodes slice
	updated_copy := make([]*Node, len(recently_updated), len(recently_updated))
	copy(updated_copy, recently_updated)

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

// Returns a unique identifying non-deterministic string for this host.
//
func generateIdentifier() string {
	bytes := make([]byte, 0, 1000)

	// Begin with the byte value of the current nano time
	//
	now_bytes, _ := time.Now().MarshalBinary()
	for _, b := range now_bytes {
		bytes = append(bytes, b)
	}

	// Append the local IP of the current machine
	//
	ipbytes := []byte{127, 0, 0, 1}
	localip, err := GetLocalIP()
	if err != nil {
		fmt.Println("WARNING: Could not resolve hostname. Using '127.0.0.1'")
	} else {
		ipbytes = []byte(localip)
	}

	for _, b := range ipbytes {
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
