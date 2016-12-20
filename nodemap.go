package blackfish

import (
	"math/rand"
	"net"
	"strconv"
	"sync"
)

type nodeMap struct {
	sync.RWMutex

	nodes map[string]*Node
}

func (m *nodeMap) init() {
	m.nodes = make(map[string]*Node)
}

// Adds a node. Returns key, value.
// Updates node heartbeat in the process.
// This is the method called by all Add* functions.
func (m *nodeMap) add(node *Node) (string, *Node, error) {
	LogInfo("Adding host:", node.Address())

	key := node.Address()

	node.Touch()
	node.heartbeats = currentHeartbeat

	m.Lock()
	m.nodes[node.Address()] = node
	m.Unlock()

	return key, node, nil
}

func (m *nodeMap) delete(node *Node) {
	m.Lock()

	delete(m.nodes, node.Address())

	m.Unlock()
}

func (m *nodeMap) contains(node *Node) bool {
	return m.containsByAddress(node.Address())
}

func (m *nodeMap) containsByAddress(address string) bool {
	m.RLock()

	_, ok := m.nodes[address]

	m.RUnlock()

	return ok
}

// Returns a pointer to the requested Node
func (m *nodeMap) getByAddress(address string) *Node {
	m.RLock()
	node, _ := m.nodes[address]
	m.RUnlock()

	return node
}

// Returns a pointer to the requested Node. If port is 0, is uses the value
// of GetListenPort(). If the Node cannot be found, this returns nil.
func (m *nodeMap) getByIP(ip net.IP, port uint16) *Node {
	if port == 0 {
		port = uint16(GetListenPort())
	}

	address := ip.String() + ":" + strconv.FormatInt(int64(port), 10)

	return m.getByAddress(address)
}

// Returns a single random node from the nodes map. If no nodes are available,
// nil is returned.
func (m *nodeMap) getRandomNode(exclude ...*Node) *Node {
	var filtered []string

	rawKeys := m.keys()

	if len(exclude) == 0 {
		filtered = rawKeys
	} else {
		filtered = make([]string, 0, len(rawKeys))

		// Build a filtered list excluding the excluded keys
	Outer:
		for _, rk := range rawKeys {
			for _, ex := range exclude {
				if rk == ex.Address() {
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
func (m *nodeMap) getRandomNodes(size int, exclude ...*Node) []*Node {
	allNodes := m.values()
	// First, shuffle the allNodes slice
	for i := range allNodes {
		j := rand.Intn(i + 1)
		allNodes[i], allNodes[j] = allNodes[j], allNodes[i]
	}

	// Copy the first size nodes that are not otherwise excluded
	filtered := make([]*Node, 0, len(allNodes))

	var c int = 0
Outer:
	for _, n := range allNodes {
		// Is the node in the excluded list?
		for _, e := range exclude {
			if n.Address() == e.Address() {
				continue Outer
			}
		}

		// Now we can append it
		filtered = append(filtered, n)
		c++

		if c >= size {
			break Outer
		}
	}

	return filtered
}

func (m *nodeMap) length() int {
	return len(m.nodes)
}

// Merges a slice of nodes into the nodes map.
// Returns a slice of the nodes that were merged or updated (or ignored for
// having exactly equal heartbeats)
func (m *nodeMap) mergeNodeLists(msgNodes []*Node) []*Node {
	mergedNodes := make([]*Node, 0, 1)

	for _, msgNode := range msgNodes {
		m.RLock()

		existingNode, ok := m.nodes[msgNode.Address()]

		m.RUnlock()

		if ok {
			// If the heartbeats are exactly equal, we don't merge the node,
			// but since we also don't want to retarnsmit it back to its source
			// we add to the slice of 'merged' nodes.
			if msgNode.heartbeats >= existingNode.heartbeats {
				mergedNodes = append(mergedNodes, existingNode)
			}

			if msgNode.heartbeats > existingNode.heartbeats {
				// We have this node in our list. Touch it to update the timestamp.
				//
				existingNode.heartbeats = msgNode.heartbeats
				existingNode.Touch()

				LogfInfo("[%s] Node exists; is now: %v\n",
					msgNode.Address(), existingNode)
			} else {
				LogfInfo("[%s] Node exists but heartbeat is older; ignoring\n",
					msgNode.Address())
			}
		} else {
			// We do not have this node in our list. Add it.
			LogInfo("New node identified:", msgNode)
			m.add(msgNode)

			mergedNodes = append(mergedNodes, msgNode)
		}
	}

	return mergedNodes
}

func (m *nodeMap) keys() []string {
	keys := make([]string, len(m.nodes))

	m.RLock()

	i := 0
	for k := range m.nodes {
		keys[i] = k
		i++
	}

	m.RUnlock()

	return keys
}

func (m *nodeMap) values() []*Node {
	values := make([]*Node, len(m.nodes))

	m.RLock()

	i := 0
	for _, v := range m.nodes {
		values[i] = v
		i++
	}

	m.RUnlock()

	return values
}
