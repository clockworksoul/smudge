/*
Copyright 2016 The Blackfish Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	key := node.Address()

	m.Lock()
	m.nodes[node.Address()] = node
	m.Unlock()

	return key, node, nil
}

func (m *nodeMap) delete(node *Node) (string, *Node, error) {
	m.Lock()

	delete(m.nodes, node.Address())

	m.Unlock()

	return node.Address(), node, nil
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
	}

	return nil
}

// Returns a slice of Node[] of from 0 to len(nodes) nodes.
// If size is < len(nodes), that many nodes are randomly chosen and
// returned.
func (m *nodeMap) getRandomNodes(size int, exclude ...*Node) []*Node {
	allNodes := m.values()

	if size == 0 {
		size = len(allNodes)
	}

	// First, shuffle the allNodes slice
	for i := range allNodes {
		j := rand.Intn(i + 1)
		allNodes[i], allNodes[j] = allNodes[j], allNodes[i]
	}

	// Copy the first size nodes that are not otherwise excluded
	filtered := make([]*Node, 0, len(allNodes))

	// Horribly inefficient. Fix this later.

	var c int
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

func (m *nodeMap) lengthWithStatus(status NodeStatus) int {
	m.RLock()

	i := 0
	for _, v := range m.nodes {
		if v.status == status {
			i++
		}
	}

	m.RUnlock()

	return i
}

func (m *nodeMap) keys() []string {
	m.RLock()

	keys := make([]string, len(m.nodes))

	i := 0
	for k := range m.nodes {
		keys[i] = k
		i++
	}

	m.RUnlock()

	return keys
}

func (m *nodeMap) values() []*Node {
	m.RLock()

	values := make([]*Node, len(m.nodes))

	i := 0
	for _, v := range m.nodes {
		values[i] = v
		i++
	}

	m.RUnlock()

	return values
}
