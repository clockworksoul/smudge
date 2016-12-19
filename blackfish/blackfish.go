package main

import (
	"blackfish"
	"flag"
	"fmt"
)

func main() {
	var node_address string
	var heartbeat_millis int
	var listen_port int
	var max_nodes_to_ping int
	var max_nodes_to_transmit int
	var millis_to_dead int
	var millis_to_stale int
	var err error

	flag.StringVar(&node_address, "node", "", "Initial node")

	flag.IntVar(&listen_port, "port",
		int(blackfish.GetListenPort()),
		"The bind port")

	flag.IntVar(&heartbeat_millis, "hbf",
		int(blackfish.GetHeartbeatMillis()),
		"The heartbeat frequency in milliseconds")

	flag.IntVar(&max_nodes_to_ping, "mping",
		int(blackfish.GetMaxNodesToPing()),
		" The maximum number of nodes to ping per heartbeat. "+
			"Setting to 0 is \"all known nodes\"")

	flag.IntVar(&max_nodes_to_transmit, "mtransmit",
		int(blackfish.GetMaxNodesToTransmit()),
		"The maximum number of nodes of data to transmit in a ping. "+
			"Setting to 0 is \"all known nodes\"")

	flag.IntVar(&millis_to_dead, "mdead",
		int(blackfish.GetDeadMillis()),
		"Millis from last update before a node is marked stale")

	flag.IntVar(&millis_to_stale, "mstale",
		int(blackfish.GetStaleMillis()),
		"Millis from last update before a node is marked dead and removed "+
			"from the nodes list")

	flag.Parse()

	blackfish.SetListenPort(listen_port)
	blackfish.SetHeartbeatMillis(heartbeat_millis)
	blackfish.SetMaxNodesToPing(max_nodes_to_ping)
	blackfish.SetMaxNodesToTransmit(max_nodes_to_transmit)
	blackfish.SetDeadMillis(millis_to_dead)
	blackfish.SetStaleMillis(millis_to_stale)

	if node_address != "" {
		node, err := blackfish.CreateNodeByAddress(node_address)

		if err == nil {
			blackfish.AddNode(node)
		}
	}

	if err == nil {
		blackfish.Begin()
	} else {
		fmt.Println(err)
	}
}
