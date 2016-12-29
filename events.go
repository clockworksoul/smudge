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

var broadcastListeners = make([]BroadcastListener, 0, 16)

var statusListeners = make([]StatusListener, 0, 16)

// BroadcastListener is the interface that must be implemented to take advantage
// of the cluster member status update notification functionality provided by
// the AddBroadcastListener() function.
type BroadcastListener interface {
	// The OnChange() function is called whenever the node is notified of any
	// change in the status of a cluster member.
	OnBroadcast(broadcast *Broadcast)
}

// AddBroadcastListener allows the submission of a BroadcastListener implementation
// whose OnChange() function will be called whenever the node is notified of any
// change in the status of a cluster member.
func AddBroadcastListener(listener BroadcastListener) {
	broadcastListeners = append(broadcastListeners, listener)
}

func doBroadcastUpdate(broadcast *Broadcast) {
	for _, sl := range broadcastListeners {
		sl.OnBroadcast(broadcast)
	}
}

// StatusListener is the interface that must be implemented to take advantage
// of the cluster member status update notification functionality provided by
// the AddStatusListener() function.
type StatusListener interface {
	// The OnChange() function is called whenever the node is notified of any
	// change in the status of a cluster member.
	OnChange(node *Node, status NodeStatus)
}

// AddStatusListener allows the submission of a StatusListener implementation
// whose OnChange() function will be called whenever the node is notified of any
// change in the status of a cluster member.
func AddStatusListener(listener StatusListener) {
	statusListeners = append(statusListeners, listener)
}

func doStatusUpdate(node *Node, status NodeStatus) {
	for _, sl := range statusListeners {
		sl.OnChange(node, status)
	}
}
