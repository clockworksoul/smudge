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
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	expectedBytes = []byte("The quick brown fox jumps over the lazy dog")

	expectedIndex = uint32(1)

	expectedEmitCounter = int8(64)

	expectedOriginIP = net.IPv4(10, 9, 8, 7)

	expectedOriginPort = uint16(1234)

	expectedLabel = fmt.Sprintf("%s:%d:%d",
		expectedOriginIP.String(),
		expectedOriginPort,
		expectedIndex)
)

func testNode() *Node {
	return &Node{
		ip:   expectedOriginIP,
		port: expectedOriginPort,
	}
}

func testBroadcast() *Broadcast {
	node := testNode()

	return &Broadcast{
		origin:      node,
		bytes:       expectedBytes,
		index:       expectedIndex,
		emitCounter: expectedEmitCounter,
	}
}

func TestBytes(t *testing.T) {
	bc := testBroadcast()
	out := bc.Bytes()

	require.Equal(t, expectedBytes, out, "Value mismatch")
}

func TestIndex(t *testing.T) {
	bc := testBroadcast()
	out := bc.Index()

	require.Equal(t, expectedIndex, out, "Value mismatch")
}

func TestLabel(t *testing.T) {
	bc := testBroadcast()
	out := bc.Label()

	require.Equal(t, expectedLabel, out, "Value mismatch")
}

func TestGetBroadcastToEmit(t *testing.T) {
	bca, bcb, bcc := testBroadcast(), testBroadcast(), testBroadcast()

	bca.emitCounter = 5
	bca.index = 1
	bcb.emitCounter = 15
	bcb.index = 2
	bcc.emitCounter = 10
	bcc.index = 3

	broadcasts.m["a"] = bca
	broadcasts.m["b"] = bcb
	broadcasts.m["c"] = bcc

	bc1 := getBroadcastToEmit()
	require.Equal(t, bcb, bc1, fmt.Sprintf("Expected %v, got %v", bcb, bc1))

	delete(broadcasts.m, "b")
	bc2 := getBroadcastToEmit()
	require.Equal(t, bcc, bc2, fmt.Sprintf("Expected %v, got %v", bcc, bc2))

	delete(broadcasts.m, "c")
	bc3 := getBroadcastToEmit()
	require.Equal(t, bca, bc3, fmt.Sprintf("Expected %v, got %v", bcc, bc3))

	delete(broadcasts.m, "a")
}

func TestBroadcastBytes(t *testing.T) {
	h := testNode()
	thisHost = h

	require.Empty(t, broadcasts.m, "Broadcasts map isn't empty")

	err := BroadcastBytes(expectedBytes)
	require.Nil(t, err, "Should have been no error!")

	bc := broadcasts.m[expectedLabel]

	require.Equal(t, expectedBytes, bc.Bytes(), "Message contents mismatch")

	delete(broadcasts.m, expectedLabel)
}

func TestBroadcastBytesTooLong(t *testing.T) {
	bytesTooLong := make([]byte, GetMaxBroadcastBytes()*2, GetMaxBroadcastBytes()*2)
	err := BroadcastBytes(bytesTooLong)
	require.NotNil(t, err, "Should have been too long!")
}

func TestReceiveBroadcast(t *testing.T) {
	bc := testBroadcast()

	require.Empty(t, broadcasts.m, "Broadcasts map isn't empty")

	receiveBroadcast(bc)

	require.True(t, broadcasts.m[expectedLabel] != nil, "Broadcast value is nil")

	receiveBroadcast(bc)

	require.True(t, len(broadcasts.m) == 1, "Added another where it shouldn't have")
}
