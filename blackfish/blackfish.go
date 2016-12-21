package main

import (
	"blackfish"
	"flag"
	"fmt"
)

func main() {
	var nodeAddress string
	var heartbeatMillis int
	var listen_port int
	var err error

	flag.StringVar(&nodeAddress, "node", "", "Initial node")

	flag.IntVar(&listen_port, "port",
		int(blackfish.GetListenPort()),
		"The bind port")

	flag.IntVar(&heartbeatMillis, "hbf",
		int(blackfish.GetHeartbeatMillis()),
		"The heartbeat frequency in milliseconds")

	flag.Parse()

	blackfish.SetListenPort(listen_port)
	blackfish.SetHeartbeatMillis(heartbeatMillis)

	if nodeAddress != "" {
		node, err := blackfish.CreateNodeByAddress(nodeAddress)

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
