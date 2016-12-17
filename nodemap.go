package blackfish

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
)

type NodeMap struct {
	sync.RWMutex

	nodes map[string]*Node
}

func (m *NodeMap) init() {
	m.nodes = make(map[string]*Node)
}

// Adds a node. Returns key, value.
// Updates node heartbeat in the process.
// This is the method called by all Add* functions.
func (m *NodeMap) Add(node *Node) (string, *Node, error) {
	fmt.Println("Adding host:", node.Address())

	key := node.Address()

	node.Touch()
	node.Heartbeats = current_heartbeat

	m.Lock()
	m.nodes[node.Address()] = node
	m.Unlock()

	return key, node, nil
}

// Given a node address ("ip:port" string), creates a new node instance
// and adds it to the map. If this address already exists in the map, this
// function replaces the existing node.
func (m *NodeMap) AddByAddress(address string) (string, *Node, error) {
	ip, port, err := parseNodeAddress(address)

	if err == nil {
		node := Node{IP: ip, Port: port, Timestamp: GetNowInMillis()}

		return m.Add(&node)
	}

	return "", nil, err
}

// Explicitly adds a node to this server's internal nodes list.
func (m *NodeMap) AddByIP(ip net.IP, port uint16) (string, *Node, error) {
	node := Node{IP: ip, Port: port, Timestamp: GetNowInMillis()}

	return m.Add(&node)
}

func (m *NodeMap) Delete(node *Node) {
	m.Lock()

	delete(m.nodes, node.Address())

	m.Unlock()
}

func (m *NodeMap) Contains(node *Node) bool {
	return m.ContainsByAddress(node.Address())
}

func (m *NodeMap) ContainsByAddress(address string) bool {
	m.RLock()

	_, ok := m.nodes[address]

	m.RUnlock()

	return ok
}

// Returns a pointer to the requested Node
func (m *NodeMap) GetByAddress(address string) *Node {
	m.RLock()
	node, _ := m.nodes[address]
	m.RUnlock()

	return node
}

// Returns a pointer to the requested Node. If port is 0, is uses the value
// of GetListenPort(). If the Node cannot be found, this returns nil.
func (m *NodeMap) GetByIP(ip net.IP, port uint16) *Node {
	if port == 0 {
		port = uint16(GetListenPort())
	}

	address := ip.String() + ":" + strconv.FormatInt(int64(port), 10)

	return m.GetByAddress(address)
}

// Returns a single random node from the nodes map. If no nodes are available,
// nil is returned.
func (m *NodeMap) GetRandom(exclude_keys ...string) *Node {
	var filtered []string

	raw_keys := m.Keys()

	if len(exclude_keys) == 0 {
		filtered = raw_keys
	} else {
		filtered = make([]string, 0, len(raw_keys))

		// Build a filtered list excluding the excluded keys
	Outer:
		for _, rk := range raw_keys {
			for _, ex := range exclude_keys {
				if rk == ex {
					continue Outer
				}
			}

			filtered = append(filtered, rk)
		}
	}

	if len(filtered) > 0 {
		// Okay, get a random index, and return the appropriate *Node
		i := rand.Intn(len(filtered))
		return m.nodes[filtered[i]]
	} else {
		return nil
	}
}

// Returns a slice of Node[] of from 0 to len(nodes) nodes.
// If size is < len(nodes), that many nodes are randomly chosen and
// returned.
func (m *NodeMap) getRandomNodes(size int, exclude ...*Node) []*Node {
	excludeMap := make(map[string]*Node)
	for _, n := range exclude {
		excludeMap[n.Address()] = n
	}

	// If size is less than the entire set of nodes, shuffle and get a subset.
	live_nodes.RLock()
	if size <= 0 || size > len(m.nodes) {
		size = len(m.nodes)
	}

	// Copy the complete nodes map into a slice
	rnodes := make([]*Node, 0, size)

	var c int
	for _, n := range m.nodes {
		// If a node is not stale...
		//
		if n.Age() < uint32(GetStaleMillis()) {
			// And isn't in the excludes map...
			//
			// if _, ok := nodes[n.Address()]; !ok {
			// We append it
			rnodes = append(rnodes, n)
			c++

			if c >= size {
				break
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

	return rnodes
}

func (m *NodeMap) Length() int {
	return len(m.nodes)
}

// Merges a slice of nodes into the nodes map.
// Returns a slice of the nodes that were merged or updated (or ignored for
// having exactly equal heartbeats)
func (m *NodeMap) mergeNodeLists(msgNodes []*Node) []*Node {
	mergedNodes := make([]*Node, 0, 1)

	for _, msgNode := range msgNodes {
		live_nodes.RLock()

		existingNode, ok := m.nodes[msgNode.Address()]

		live_nodes.RUnlock()

		if ok {
			// If the heartbeats are exactly equal, we don't merge the node,
			// but since we also don't want to retarnsmit it back to its source
			// we add to the slice of 'merged' nodes.
			if msgNode.Heartbeats >= existingNode.Heartbeats {
				mergedNodes = append(mergedNodes, existingNode)
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
			m.Add(msgNode)

			mergedNodes = append(mergedNodes, msgNode)
		}
	}

	return mergedNodes
}

func (m *NodeMap) Keys() []string {
	keys := make([]string, len(m.nodes))

	live_nodes.RLock()

	i := 0
	for k := range m.nodes {
		keys[i] = k
		i++
	}

	live_nodes.RUnlock()

	return keys
}

func (m *NodeMap) Values() []*Node {
	values := make([]*Node, len(m.nodes))

	live_nodes.RLock()

	i := 0
	for _, v := range m.nodes {
		values[i] = v
		i++
	}

	live_nodes.RUnlock()

	return values
}
