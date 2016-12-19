package blackfish

import (
	"fmt"
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
	fmt.Println("Adding host:", node.Address())

	key := node.Address()

	node.Touch()
	node.Heartbeats = current_heartbeat

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
func (m *nodeMap) getRandom(exclude_keys ...string) *Node {
	var filtered []string

	raw_keys := m.keys()

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
func (m *nodeMap) getRandomNodes(size int, exclude ...*Node) []*Node {
	all_nodes := m.values()
	// First, shuffle the all_nodes slice
	for i := range all_nodes {
		j := rand.Intn(i + 1)
		all_nodes[i], all_nodes[j] = all_nodes[j], all_nodes[i]
	}

	// Copy the first size nodes that are not otherwise excluded
	filtered_nodes := make([]*Node, 0, len(all_nodes))

	var c int = 0
AppendLoop:
	for _, n := range all_nodes {
		// Is the node in the excluded list?
		for _, e := range exclude {
			if n.Address() == e.Address() {
				continue AppendLoop
			}
		}

		// Now we can append it
		filtered_nodes = append(filtered_nodes, n)
		c++

		if c >= size {
			break AppendLoop
		}
	}

	return filtered_nodes
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
			m.add(msgNode)

			mergedNodes = append(mergedNodes, msgNode)
		}
	}

	return mergedNodes
}

func (m *nodeMap) keys() []string {
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

func (m *nodeMap) values() []*Node {
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
