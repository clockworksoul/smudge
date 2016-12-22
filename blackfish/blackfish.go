/*
Copyright 2015 The Blackfish Authors.

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

package main

import (
	"blackfish"
	"flag"
	"fmt"
)

func main() {
	var nodeAddress string
	var heartbeatMillis int
	var listenPort int
	var err error

	flag.StringVar(&nodeAddress, "node", "", "Initial node")

	flag.IntVar(&listenPort, "port",
		int(blackfish.GetListenPort()),
		"The bind port")

	flag.IntVar(&heartbeatMillis, "hbf",
		int(blackfish.GetHeartbeatMillis()),
		"The heartbeat frequency in milliseconds")

	flag.Parse()

	blackfish.SetLogThreshold(blackfish.LogInfo)

	blackfish.SetListenPort(listenPort)
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
