/*
Copyright 2016 The Smudge Authors.

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

package smudge

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
)

// All known nodes, living and dead. Dead nodes are pinged (far) less often,
// and are eventually removed
var knownNodes = nodeMap{}

// All nodes that have been updated "recently", living and dead
var updatedNodes = nodeMap{}

var deadNodeRetries = struct {
	sync.RWMutex
	m map[string]*deadNodeCounter
}{m: make(map[string]*deadNodeCounter)}

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
		return CreateNodeByIP(ip, port)
	}

	return nil, err
}

// CreateNodeByIP will create and return a new node when supplied with an
// IP address and port number. This doesn't add the node to the list of live
// nodes; use AddNode().
func CreateNodeByIP(ip net.IP, port uint16) (*Node, error) {
	node := Node{
		ip:         ip,
		port:       port,
		timestamp:  GetNowInMillis(),
		pingMillis: PingNoData,
	}

	return &node, nil
}

// GetLocalIP queries the host interface to determine the local IP address of this
// machine. If a local IP address cannot be found, then nil is returned. Local IPv6
// address takes presedence over a local IPv4 address. If the query to the underlying
// OS fails, an error is returned.
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
			} else {
				ip = ipnet.IP
				break
			}
		}
	}

	return ip, nil
}

// AllNodes will return a list of all nodes known at the time of the request,
// including nodes that have been marked as "dead" but haven't yet been
// removed from the registry.
func AllNodes() []*Node {
	return knownNodes.values()
}

// HealthyNodes will return a list of all nodes known at the time of the
// request with a healthy status.
func HealthyNodes() []*Node {
	values := knownNodes.values()
	filtered := make([]*Node, 0, len(values))

	for _, v := range values {
		if v.Status() == StatusAlive {
			filtered = append(filtered, v)
		}
	}

	return filtered
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
	updateNodeStatus(node, status, node.heartbeat)
}

/******************************************************************************
 * Private functions (for internal use only)
 *****************************************************************************/

func getRandomUpdatedNodes(size int, exclude ...*Node) []*Node {
	updatedNodesCopy := nodeMap{}
	updatedNodesCopy.init()

	// Prune nodes with emit counters of 0 (or less) from the map. Any
	// others we copy into a secondary nodemap.
	for _, n := range updatedNodes.values() {
		if n.emitCounter <= 0 {
			logDebug("Removing", n.Address(), "from recently updated list")
			updatedNodes.delete(n)
		} else {
			updatedNodesCopy.add(n)
		}
	}

	// Exclude the exclusions
	for _, ex := range exclude {
		updatedNodesCopy.delete(ex)
	}

	// Put the newest nodes on top.
	updatedNodesSlice := updatedNodesCopy.values()
	sort.Sort(byNodeEmitCounter(updatedNodesSlice))

	// Grab and return the top N
	if size > len(updatedNodesSlice) {
		size = len(updatedNodesSlice)
	}

	return updatedNodesSlice[:size]
}

func parseNodeAddress(hostAndMaybePort string) (net.IP, uint16, error) {
	var host string
	var ip net.IP
	var port uint16
	var err error

	ip = net.ParseIP(hostAndMaybePort)
	port = uint16(GetListenPort())

	host, sport, err := net.SplitHostPort(hostAndMaybePort)

	if err == nil {
		if sport != "" {
			p, e := strconv.ParseUint(sport, 10, 16)
			port = uint16(p)
			err = e
		}

		ip = net.ParseIP(host)
	} else {
		err = nil
		ip = net.ParseIP(hostAndMaybePort)
		port = uint16(GetListenPort())

		if host == "" {
			host = hostAndMaybePort
		}
	}

	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil {
			return ip, port, err
		}

		for _, i := range ips {
			if !i.IsLoopback() {
				if GetListenIP().To4() != nil && i.To4() != nil {
					ip = i
					break
				} else if GetListenIP().To4() == nil && i.To4() == nil {
					ip = i
					break
				}
			}
		}

		if ip == nil {
			err = fmt.Errorf("Could not parse the address of node %s", hostAndMaybePort)
		}
	}

	return ip, port, err
}

// UpdateNodeStatus assigns a new status for the specified node and adds it to
// the list of recently updated nodes. If the status is StatusDead, then the
// node will be moved from the live nodes list to the dead nodes list.
func updateNodeStatus(node *Node, status NodeStatus, heartbeat uint32) {
	if node.status != status {
		if heartbeat < node.heartbeat {
			logfWarn("Decreasing known node heartbeat value from %d to %d\n",
				node.heartbeat,
				heartbeat)
		}

		node.timestamp = GetNowInMillis()
		node.status = status
		node.emitCounter = int8(emitCount())
		node.heartbeat = heartbeat

		// If this isn't in the recently updated list, add it.
		if !updatedNodes.contains(node) {
			updatedNodes.add(node)
		}

		if status != StatusDead {
			deadNodeRetries.Lock()
			delete(deadNodeRetries.m, node.Address())
			deadNodeRetries.Unlock()
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

type deadNodeCounter struct {
	retry          int
	retryCountdown int
}

// byNodeEmitCounter implements sort.Interface for []*Node based on
// the emitCounter field.
type byNodeEmitCounter []*Node

func (a byNodeEmitCounter) Len() int {
	return len(a)
}

func (a byNodeEmitCounter) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a byNodeEmitCounter) Less(i, j int) bool {
	return a[i].emitCounter > a[j].emitCounter
}
