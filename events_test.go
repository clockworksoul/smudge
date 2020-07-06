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
	"testing"

	"github.com/stretchr/testify/require"
)

type TestBroadcastListener struct {
	broadcast *Broadcast
}

func (l *TestBroadcastListener) OnBroadcast(broadcast *Broadcast) {
	l.broadcast = broadcast
}

func TestBroadcastListeners(t *testing.T) {
	require.Empty(t, broadcastListeners.s)

	l := &TestBroadcastListener{}

	AddBroadcastListener(l)
	require.Equal(t, 1, len(broadcastListeners.s))

	require.Nil(t, l.broadcast)

	bc := testBroadcast()
	doBroadcastUpdate(bc)

	require.NotNil(t, l.broadcast)

	require.Equal(t, bc, l.broadcast)
}

type TestStatusListener struct {
	node   *Node
	status NodeStatus
}

func (l *TestStatusListener) OnChange(node *Node, status NodeStatus) {
	l.node = node
	l.status = status
}

func TestStatusListeners(t *testing.T) {
	require.Empty(t, statusListeners.s)

	l := &TestStatusListener{}

	AddStatusListener(l)
	require.Equal(t, 1, len(statusListeners.s))

	require.Nil(t, l.node)
	require.Equal(t, StatusUnknown, l.status)

	n := testNode()
	s := StatusAlive

	doStatusUpdate(n, s)

	require.Equal(t, s, l.status)
	require.Equal(t, n, l.node)
}
