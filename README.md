# Blackfish

[![GoDoc](https://godoc.org/github.com/ClockworkSoul/blackfish?status.svg)](https://godoc.org/github.com/ClockworkSoul/blackfish)

## Introduction 
Blackfish is a minimalist Go implementation of the [SWIM](https://www.cs.cornell.edu/~asdas/research/dsn02-swim.pdf) (Scalable Weakly-consistent Infection-style Membership) protocol for node membership, status dissemination, and failure detection developed at Cornell University by Motivala, et al. It isn't a distributed data store in its own right, but rather a framework intended to facilitate the construction of such systems.

It was conceived with a space-sensitive systems (mobile, IOT, containers) in mind, and therefore was developed with a minimalist philosophy of doing a few things well. As such, its feature set is relatively small and limited to functionality around adding and removing nodes and detecting status changes on the cluster.

Complete documentation is available from [the associated Godoc](https://godoc.org/github.com/ClockworkSoul/blackfish).

## Features
* Uses gossip (i.e., epidemic) protocol for dissemination, the latency of which grows logarithmically with the number of members.
* Low-bandwidth UDP-based failure detection and status dissemination.
* Imposes a constant message load per group member, regardless of the number of members.
* Member status changes are eventually detected by all non-faulty members of the cluster (strong completeness).
* Various knobs and levers to tweak ping and dissemination behavior, settable via the API or environment variables.

### Coming soon!
* Support for multicast announcement and recruitment.
* Adaptive timeouts (defined as the 99th percentile of all recently seen responses; currently hard-coded at 150ms).

### Deviations from [Motivala, et al](https://www.cs.cornell.edu/~asdas/research/dsn02-swim.pdf)

* Dead nodes are not immediately removed, but are instead periodically re-tried (with exponential backoff) for a time before finally being removed.

## How to use
To use the code, you simply specify a few configuration options (or use the defaults), create and add a node status change listener, and call the `blackfish.Begin()` function.


### Configuring the node with environment variables
Perhaps the simplest way of directing the behavior of the SWIM driver is by setting the appropriate system environment variables, which is useful when making use of Blackfish inside of a container.

The following variables and their default values are as follows:

```
Variable                   | Default | Description
-------------------------- | ------- | -------------------------------
BLACKFISH_HEARTBEAT_MILLIS |     500 | Milliseconds between heartbeats
BLACKFISH_LISTEN_PORT      |    9999 | UDP port to listen on 
```

### Configuring the node with environment variables
If you prefer to direct the behavior of the service using the API, the calls are relatively straight-forward. Note that setting the application properties using this method overrides the behavior of environment variables.

```
blackfish.SetListenPort(9999)
blackfish.SetHeartbeatMillis(500)
```

### Creating and adding a status change listener
Creating a status change listener is very straight-forward. 

Simply: 

```
package main

import "blackfish"
import "fmt"

type MyListener struct {
	blackfish.StatusListener
}

func (m MyListener) OnChange(node *blackfish.Node, status blackfish.NodeStatus) {
	fmt.Printf("Node %s is now status %s\n", node.Address(), status)
}

func main() {
	blackfish.AddStatusListener(MyListener{})
}
```

### Adding a new member to the "known nodes" list
Adding a new member to your known nodes list will also make that node aware of the adding server. Note that because this package doesn't yet support multicast notifications, at this time to join an existing cluster you must use this method to add at least one of that cluster's healthy member nodes.

```
node, err := blackfish.CreateNodeByAddress("localhost:10000")
if err == nil {
    blackfish.AddNode(node)
}
```


### Starting the server
Once everything else is done, starting the server is trivial.

Simply call: `blackfish.Begin()`

### Everything in one place

```
package main

import "blackfish"
import "fmt"

type MyListener struct {
	blackfish.StatusListener
}

func (m MyListener) OnChange(node *blackfish.Node, status blackfish.NodeStatus) {
	fmt.Printf("Node %s is now status %s\n", node.Address(), status)
}

func main() {
	heartbeatMillis := 500
	listenPort := 9999

	// Set configuration options
	blackfish.SetListenPort(listenPort)
	blackfish.SetHeartbeatMillis(heartbeatMillis)

	// Add the listener
	blackfish.AddStatusListener(MyListener{})

	// Add a new remote node. Currently, to join an existing cluster you must
	// add at least one of its healthy member nodes.
	node, err := blackfish.CreateNodeByAddress("localhost:10000")
	if err == nil {
		blackfish.AddNode(node)
	}

	// Start the server!
	blackfish.Begin()
}
```