package blackfish

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

// Currenty active nodes.
var live_nodes NodeMap

// Recently dead nodes. Periodically a random dead node will be allowed to
// rejoin the living.
var dead_nodes []Node

func init() {
	live_nodes = NodeMap{}
	live_nodes.init()

	dead_nodes = make([]Node, 0, 64)
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

// Loops through the nodes map and removes nodes that haven't been heard
// from in > GetDeadMillis() milliseconds.
func PruneDeadFromList() {
	live_nodes.Lock()

	for k, n := range live_nodes.nodes {
		if n.Age() > uint32(GetDeadMillis()) {
			fmt.Printf("Node removed [%d > %d]: %v\n", n.Age(), GetDeadMillis(), n)

			delete(live_nodes.nodes, k)

			dead_nodes = append(dead_nodes, *n)
		}
	}

	live_nodes.Unlock()
}

// Returns a random dead node to the live_nodes map.
func ResurrectDeadNode() {
	if len(dead_nodes) == 1 {
		live_nodes.Add(dead_nodes[0])
		dead_nodes = make([]Node, 0, 64)
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
	live_nodes.RLock()

	if size <= 0 || size > live_nodes.Length() {
		size = live_nodes.Length()
	}

	// Copy the complete nodes map into a slice
	rnodes := make([]Node, 0, size)

	var c int
	for _, n := range live_nodes.nodes {
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

	live_nodes.RUnlock()

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
