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
	var err error

	flag.StringVar(&node_address, "node", "", "Initial node")

	flag.IntVar(&listen_port, "port",
		int(blackfish.GetListenPort()),
		"The bind port")

	flag.IntVar(&heartbeat_millis, "hbf",
		int(blackfish.GetHeartbeatMillis()),
		"The heartbeat frequency in milliseconds")

	flag.Parse()

	blackfish.SetListenPort(listen_port)
	blackfish.SetHeartbeatMillis(heartbeat_millis)

	if node_address != "" {
		node, err := blackfish.CreateNodeByAddress(node_address)

		if err == nil {
			blackfish.UpdateNodeStatus(node, blackfish.STATUS_ALIVE)
			blackfish.AddNode(node)
		}
	}

	if err == nil {
		blackfish.Begin()
	} else {
		fmt.Println(err)
	}
}
