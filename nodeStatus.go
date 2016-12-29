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

// NodeStatus represents the believed status of a member node.
type NodeStatus byte

const (
	// StatusUnknown is the default node status of newly-created nodes.
	StatusUnknown NodeStatus = iota

	// StatusAlive indicates that a node is alive and healthy.
	StatusAlive

	// StatusDead indicatates that a node is dead and no longer healthy.
	StatusDead

	// StatusForwardTo is a pseudo status used by message to indicate
	// the target of a ping request.
	StatusForwardTo
)

func (s NodeStatus) String() string {
	switch s {
	case StatusUnknown:
		return "UNKNOWN"
	case StatusAlive:
		return "ALIVE"
	case StatusDead:
		return "DEAD"
	case StatusForwardTo:
		return "FORWARD_TO"
	default:
		return "UNDEFINED"
	}
}
