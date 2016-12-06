package main

import (
	"flag"
	"fleacircus"
	"fmt"
)

var manager fleacircus.Manager

func main() {
	var port int
	var node string

	flag.IntVar(&port, "port", int(fleacircus.DEFAULT_PORT), "The bind port")
	flag.StringVar(&node, "node", "", "Initial node")

	flag.Parse()

	manager = fleacircus.Manager{}
	manager.AddNode(node)
	manager.SetPort(port)
	manager.Begin()

	fmt.Println("Done")
}
