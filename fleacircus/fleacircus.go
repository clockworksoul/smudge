package main

import (
	"flag"
	"fleacircus"
)

func main() {
	var node string
	var heartbeat_millis int
	var listen_port int
	var max_nodes_to_ping int
	var max_nodes_to_transmit int
	var millis_to_dead int
	var millis_to_stale int

	flag.StringVar(&node, "node", "", "Initial node")

	flag.IntVar(&listen_port, "port",
		int(fleacircus.GetListenPort()),
		"The bind port")

	flag.IntVar(&heartbeat_millis, "hbf",
		int(fleacircus.GetHeartbeatMillis()),
		"The heartbeat frequency in milliseconds")

	flag.IntVar(&max_nodes_to_ping, "mping",
		int(fleacircus.GetMaxNodesToPing()),
		" The maximum number of nodes to ping per heartbeat. "+
			"Setting to 0 is \"all known nodes\"")

	flag.IntVar(&max_nodes_to_transmit, "mtransmit",
		int(fleacircus.GetMaxNodesToTransmit()),
		"The maximum number of nodes of data to transmit in a ping. "+
			"Setting to 0 is \"all known nodes\"")

	flag.IntVar(&millis_to_dead, "mdead",
		int(fleacircus.GetDeadMillis()),
		"Millis from last update before a node is marked stale")

	flag.IntVar(&millis_to_stale, "mstale",
		int(fleacircus.GetStaleMillis()),
		"Millis from last update before a node is marked dead and removed "+
			"from the nodes list")

	flag.Parse()

	fleacircus.SetListenPort(listen_port)
	fleacircus.SetHeartbeatMillis(heartbeat_millis)
	fleacircus.SetMaxNodesToPing(max_nodes_to_ping)
	fleacircus.SetMaxNodesToTransmit(max_nodes_to_transmit)
	fleacircus.SetDeadMillis(millis_to_dead)
	fleacircus.SetStaleMillis(millis_to_stale)

	if node != "" {
		fleacircus.AddNode(node)
	}

	fleacircus.Begin()
}
