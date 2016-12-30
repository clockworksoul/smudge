# Smudge

[![GoDoc](https://godoc.org/github.com/clockworksoul/smudge?status.svg)](https://godoc.org/github.com/clockworksoul/smudge)
[![Build Status](https://travis-ci.org/clockworksoul/smudge.svg?branch=master)](https://travis-ci.org/clockworksoul/smudge)
[![Go Report Card](https://goreportcard.com/badge/github.com/clockworksoul/smudge)](https://goreportcard.com/report/github.com/clockworksoul/smudge)

<img src="https://github.com/clockworksoul/smudge/raw/master/logo/logo.png" width="150">

## Introduction 
Smudge is a minimalist Go implementation of the [SWIM](https://www.cs.cornell.edu/~asdas/research/dsn02-swim.pdf) (Scalable Weakly-consistent Infection-style Membership) protocol for cluster node membership, status dissemination, and failure detection developed at Cornell University by Motivala, et al. It isn't a distributed data store in its own right, but rather a framework intended to facilitate the construction of such systems.

Smudge also extends the standard SWIM protocol so that in addition to the standard membership status functionality it also allows the transmission of broadcasts containing a small amount (256 bytes) of arbitrary content to all present healthy members. This maximum is related to the maximum safe packet size imposed on UDP packet size (RFC 791, RFC 2460). We recognize that some systems allow larger packets, however, and while that can risk fragmentation and dropped packets this value is configurable.

Smudge was conceived with a space-sensitive systems (mobile, IOT, containers) in mind, and therefore was developed with a minimalist philosophy of doing a few things well. As such, its feature set is relatively small and limited to functionality around adding and removing nodes and detecting status changes on the cluster.

Complete documentation is available from [the associated Godoc](https://godoc.org/github.com/clockworksoul/smudge).


## Features
* Uses gossip (i.e., epidemic) protocol for dissemination, the latency of which grows logarithmically with the number of members.
* Low-bandwidth UDP-based failure detection and status dissemination.
* Imposes a constant message load per group member, regardless of the number of members.
* Member status changes are eventually detected by all non-faulty members of the cluster (strong completeness).
* Supports transmission of short (256 byte) broadcasts that are propagated at most once to all present, healthy members.


## Known issues
* Broadcasts are limited to 256 bytes.
* No WAN support: only local-network, private IPs are supported.
* No multicast discovery.


### Deviations from [Motivala, et al](https://www.cs.cornell.edu/~asdas/research/dsn02-swim.pdf)

* Dead nodes are not immediately removed, but are instead periodically re-tried (with exponential backoff) for a time before finally being removed.
* Smudge allows the transsion of short, arbitrary-content broadcasts to all healthy nodes.


## How to use
To use the code, you simply specify a few configuration options (or use the defaults), create and add a node status change listener, and call the `smudge.Begin()` function.


### Configuring the node with environment variables
Perhaps the simplest way of directing the behavior of the SWIM driver is by setting the appropriate system environment variables, which is useful when making use of Smudge inside of a container.

The following variables and their default values are as follows:

```
Variable                   | Default | Description
-------------------------- | ------- | -------------------------------
SMUDGE_HEARTBEAT_MILLIS    |     500 | Milliseconds between heartbeats
SMUDGE_INITIAL_HOSTS       |         | Comma-delimmited list of known members as IP or IP:PORT.
SMUDGE_LISTEN_PORT         |    9999 | UDP port to listen on
SMUDGE_MAX_BROADCAST_BYTES |     256 | Maximum byte length of broadcast payloads
```


### Configuring the node with environment variables
If you prefer to direct the behavior of the service using the API, the calls are relatively straight-forward. Note that setting the application properties using this method overrides the behavior of environment variables.

```
smudge.SetListenPort(9999)
smudge.SetHeartbeatMillis(500)
smudge.SetMaxBroadcastBytes(512)
```


### Creating and adding a status change listener
Creating a status change listener is very straight-forward. 

```
type MyStatusListener struct {
	smudge.StatusListener
}

func (m MyStatusListener) OnChange(node *smudge.Node, status smudge.NodeStatus) {
	fmt.Printf("Node %s is now status %s\n", node.Address(), status)
}

func main() {
	smudge.AddStatusListener(MyStatusListener{})
}
```


### Creating and adding a broadcast listener
Adding a broadcast listener is very similar to creating a status listener: 

```
type MyBroadcastListener struct {
	smudge.BroadcastListener
}

func (m MyBroadcastListener) OnBroadcast(b *smudge.Broadcast) {
	fmt.Printf("Received broadcast from %v: %s\n",
		b.Origin().Address(),
		string(b.Bytes()))
}

func main() {
	smudge.AddBroadcastListener(MyBroadcastListener{})
}
```


### Adding a new member to the "known nodes" list
Adding a new member to your known nodes list will also make that node aware of the adding server. Note that because this package doesn't yet support multicast notifications, at this time to join an existing cluster you must use this method to add at least one of that cluster's healthy member nodes.

```
node, err := smudge.CreateNodeByAddress("localhost:10000")
if err == nil {
    smudge.AddNode(node)
}
```


### Starting the server
Once everything else is done, starting the server is trivial.

Simply call: `smudge.Begin()`


### Transmitting a broadcast
To transmit a broadcast to all healthy nodes currenty in the cluster you can use one of the [`BroadcastBytes(bytes []byte)`](https://godoc.org/github.com/clockworksoul/smudge#BroadcastBytes) or [`BroadcastString(str string)`](https://godoc.org/github.com/clockworksoul/smudge#BroadcastString) functions.

Be aware of the following caveats:
* Attempting to send a broadcast before the server has been started with cause a panic.
* The broadcast _will not_ be received by the originating member; `BroadcastListener`s on the originating member will not be triggered.
* Nodes that join the cluster after the broadcast has been fully propagated will not receive broadcast; nodes that join after the initial transmission but before complete proagation may or may not receive the broadcast.


### Getting a list of nodes
The [`AllNodes()`](https://godoc.org/github.com/clockworksoul/smudge#AllNodes) can be used to give access to all known nodes. [`HealthyNodes()`](https://godoc.org/github.com/clockworksoul/smudge#HealthyNodes) works similarly, but returns only healthy nodes (defined as nodes with a [status](https://godoc.org/github.com/clockworksoul/smudge#NodeStatus) of "alive").


### Everything in one place

```
package main

import "github.com/clockworksoul/smudge"
import "fmt"

type MyStatusListener struct {
	smudge.StatusListener
}

func (m MyStatusListener) OnChange(node *smudge.Node, status smudge.NodeStatus) {
	fmt.Printf("Node %s is now status %s\n", node.Address(), status)
}

type MyBroadcastListener struct {
	smudge.BroadcastListener
}

func (m MyBroadcastListener) OnBroadcast(b *smudge.Broadcast) {
	fmt.Printf("Received broadcast from %s: %s\n",
		b.Origin().Address(),
		string(b.Bytes()))
}

func main() {
	heartbeatMillis := 500
	listenPort := 9999

	// Set configuration options
	smudge.SetListenPort(listenPort)
	smudge.SetHeartbeatMillis(heartbeatMillis)

	// Add the status listener
	smudge.AddStatusListener(MyStatusListener{})

	// Add the broadcast listener
	smudge.AddBroadcastListener(MyBroadcastListener{})

	// Add a new remote node. Currently, to join an existing cluster you must
	// add at least one of its healthy member nodes.
	node, err := smudge.CreateNodeByAddress("localhost:10000")
	if err == nil {
		smudge.AddNode(node)
	}

	// Start the server!
	smudge.Begin()
}
```