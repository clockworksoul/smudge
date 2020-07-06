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

func TestAddNode(t *testing.T) {
	// Start empty
	require.Empty(t, knownNodes.nodes)

	n := testNode()
	n.status = StatusUnknown

	// Adding a nodes with an unknown status sets the status to ALIVE
	AddNode(n)
	require.Equal(t, StatusAlive, n.status)

	key := fmt.Sprintf("%s:%d", n.ip.String(), n.port)

	// Also, it adds exactly one node: this one.
	require.Equal(t, 1, len(knownNodes.values()))
	require.NotNil(t, knownNodes.nodes[key])
	require.Equal(t, n, knownNodes.nodes[key])

	// Adding again is a no-op
	AddNode(n)
	require.Equal(t, 1, len(knownNodes.values()))

	t.Log(n.status)
}

func TestCreateNodeByAddress(t *testing.T) {
	s := "127.0.0.1:1234"

	expected := &Node{
		ip:   net.IPv4(127, 0, 0, 1),
		port: 1234,
	}

	node, err := CreateNodeByAddress(s)
	require.Nil(t, err)
	require.Equal(t, expected.ip, node.ip)
	require.Equal(t, expected.port, node.port)
}

func TestIPv4(t *testing.T) {
	s := "127.0.0.1"
	ip, port, err := parseNodeAddress(s)

	if err != nil {
		t.Error("Error should be nil but was:", err)
	}

	if ip.String() != s {
		t.Errorf("Expected %s but found %s\n", s, ip.String())
	}

	if port != 9999 {
		t.Errorf("Expected the port to be 9999 but found %d\n", port)
	}
}

func TestIPv4WithPort(t *testing.T) {
	s := "127.0.0.1:80"
	ip, port, err := parseNodeAddress(s)

	if err != nil {
		t.Error("Error should be nil but was:", err)
	}

	if ip.String() != "127.0.0.1" {
		t.Errorf("Expected %s but found %s\n", s, ip.String())
	}

	if port != 80 {
		t.Errorf("Expected the port to be 80 but found %d\n", port)
	}
}

func TestIPv6(t *testing.T) {
	s := "fd02:6b8:b010:9020:1::2"
	ip, port, err := parseNodeAddress(s)

	if err != nil {
		t.Error("Error should be nil but was:", err)
	}

	if ip.String() != s {
		t.Errorf("Expected %s but found %s\n", s, ip.String())
	}

	if port != 9999 {
		t.Errorf("Expected the port to be 9999 but found %d\n", port)
	}
}

func TestIPv6WithPort(t *testing.T) {
	s := "[fd02:6b8:b010:9020:1::2]:80"
	ip, port, err := parseNodeAddress(s)

	if err != nil {
		t.Error("Error should be nil but was:", err)
	}

	if ip.String() != "fd02:6b8:b010:9020:1::2" {
		t.Errorf("Expected fd02:6b8:b010:9020:1::2 but found %s\n", ip.String())
	}

	if port != 80 {
		t.Errorf("Expected the port to be 80 but found %d\n", port)
	}
}

//func TestHostname(t *testing.T) {
//	s := "localhost"
//	ip, port, err := parseNodeAddress(s)
//
//	if err != nil {
//		t.Error("Error should be nil but was:", err)
//	}
//
//	if ip.String() != "127.0.0.1" {
//		t.Errorf("Expected %s but found %s\n", s, ip.String())
//	}
//
//	if port != 9999 {
//		t.Errorf("Expected the port to be 9999 but found %d\n", port)
//	}
//}
//
//func TestHostnameWithPort(t *testing.T) {
//	s := "localhost:80"
//	ip, port, err := parseNodeAddress(s)
//
//	if err != nil {
//		t.Error("Error should be nil but was:", err)
//	}
//
//	if ip.String() != "127.0.0.1" {
//		t.Errorf("Expected %s but found %s\n", s, ip.String())
//	}
//
//	if port != 80 {
//		t.Errorf("Expected the port to be 80 but found %d\n", port)
//	}
//}
//
//func TestHostnameErr(t *testing.T) {
//	SetListenIP(net.ParseIP("fd02:6b8:b010:9020:1::2"))
//
//	s := "localhost"
//	_, _, err := parseNodeAddress(s)
//
//	if err == nil {
//		t.Error("Error should not have been be nil")
//	}
//
//	SetListenIP(net.ParseIP("127.0.0.1"))
//}
