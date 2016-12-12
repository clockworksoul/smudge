package blackfish

import (
	"fmt"
	"os"
	"strconv"
)

/**
 * Provides a series of methods and constants that revolve around the getting
 * (or programmatically setting/overriding) environmental properties, returning
 * default values if not set.
 */

const (
	// The default heartbeat frequency in milliseconds for each node.
	ENV_HEARTBEAT_MILLIS         = "FLEACIRCUS_HEARTBEAT_MILLIS"
	DEFAULT_HEARTBEAT_MILLIS int = 500

	// The port to listen on by default.
	ENV_LISTEN_PORT     string = "FLEACIRCUS_LISTEN_PORT"
	DEFAULT_LISTEN_PORT int    = 9999

	// The default number of nodes to ping per heartbeat.
	// Setting to 0 is "all known nodes". This is not wise
	// In very large systems.
	ENV_MAX_NODES_TO_PING         = "FLEACIRCUS_MAX_NODES_TO_PING"
	DEFAULT_MAX_NODES_TO_PING int = 100

	// The default number of nodes of data to transmit in a ping.
	// Setting to 0 is "all known nodes". This is not wise
	// In very large systems.
	ENV_MAX_NODES_TO_TRANSMIT         = "FLEACIRCUS_MAX_NODES_TO_TRANSMIT"
	DEFAULT_MAX_NODES_TO_TRANSMIT int = 1000

	// Millis from last update before a node is marked dead and removed
	// from the nodes list.
	ENV_MILLIS_TO_DEAD         = "FLEACIRCUS_MILLIS_TO_DEAD"
	DEFAULT_MILLIS_TO_DEAD int = 5000

	// Millis from last update before a node is marked stale.
	ENV_MILLIS_TO_STALE         = "FLEACIRCUS_MILLIS_TO_STALE"
	DEFAULT_MILLIS_TO_STALE int = 2000
)

var heartbeat_millis int

var listen_address string

var listen_port int

var max_nodes_to_ping int

var max_nodes_to_transmit int

var millis_to_dead int

var millis_to_stale int

/**
 * The port to listen on.
 */
func GetListenPort() int {
	if listen_port == 0 {
		listen_port = getIntVar(ENV_LISTEN_PORT, DEFAULT_LISTEN_PORT)
	}

	return listen_port
}

/**
 * The port to listen on.
 */
func SetListenPort(p int) {
	if p == 0 {
		listen_port = DEFAULT_LISTEN_PORT
	} else {
		listen_port = p
	}
}

/**
 * The address of the host where the process lives.
 * If not set, the node will attempt to ask other nodes what its address is.
 */
func GetListenAddress() string {
	return listen_address
}

/**
 * The address of the host where the process lives.
 * If not set, the node will attempt to ask other nodes what its address is.
 */
func SetListenAddress(s string) {
	fmt.Println("Listen address:", s)
	listen_address = s
}

/**
 * The heartbeat frequency in milliseconds for each node.
 */
func GetHeartbeatMillis() int {
	if heartbeat_millis == 0 {
		heartbeat_millis = getIntVar(ENV_HEARTBEAT_MILLIS, DEFAULT_HEARTBEAT_MILLIS)
	}

	return heartbeat_millis
}

/**
 * The heartbeat frequency in milliseconds for each node.
 */
func SetHeartbeatMillis(val int) {
	heartbeat_millis = val
}

/**
 * The port to listen on.
 */
func GetMaxNodesToPing() int {
	if max_nodes_to_ping == 0 {
		max_nodes_to_ping = getIntVar(ENV_MAX_NODES_TO_PING, DEFAULT_MAX_NODES_TO_PING)
	}

	return max_nodes_to_ping
}

/**
 * The maximum number of nodes to ping per heartbeat. Setting to 0 is "all
 * known nodes". This is not wise in very large systems.
 */
func SetMaxNodesToPing(val int) {
	max_nodes_to_ping = val
}

/**
 * The maximum number of nodes to ping per heartbeat. Setting to 0 is "all
 * known nodes". This is not wise in very large systems.
 */
func GetMaxNodesToTransmit() int {
	if max_nodes_to_transmit == 0 {
		max_nodes_to_transmit = getIntVar(ENV_MAX_NODES_TO_TRANSMIT, DEFAULT_MAX_NODES_TO_TRANSMIT)
	}

	return max_nodes_to_transmit
}

/**
 * The maximum number of nodes of data to transmit in a ping. Setting to 0 is
 * "all known nodes". This is not wisevin very large systems.
 */
func SetMaxNodesToTransmit(val int) {
	max_nodes_to_ping = val
}

/**
 * Millis from last update before a node is marked dead and removed from the
 * nodes list.
 */
func GetDeadMillis() int {
	if millis_to_dead == 0 {
		millis_to_dead = getIntVar(ENV_MILLIS_TO_DEAD, DEFAULT_MILLIS_TO_DEAD)
	}

	return millis_to_dead
}

/**
 * Millis from last update before a node is marked dead and removed from the
 * nodes list.
 */
func SetDeadMillis(val int) {
	millis_to_dead = val
}

/**
 * Millis from last update before a node is marked dead and removed from
 * the nodes list.
 */
func GetStaleMillis() int {
	if millis_to_stale == 0 {
		millis_to_stale = getIntVar(ENV_MILLIS_TO_STALE, DEFAULT_MILLIS_TO_STALE)
	}

	return millis_to_stale
}

/**
 * Millis from last update before a node is marked dead and removed from
 * the nodes list.
 */
func SetStaleMillis(val int) {
	millis_to_stale = val
}

/**
 * Gets an environmental variable "key". If it does not exist, "defval" is
 * returned; if it does, it attempts to convert to an integer, returning
 * "defval" is it fails.
 */
func getIntVar(key string, defval int) int {
	var valueString string = os.Getenv(key)
	var valueInt int = defval

	if valueString != "" {
		i, err := strconv.Atoi(key)

		if err != nil {
			fmt.Printf("WARNING! Failed to parse env property %s: %s is not "+
				"an integer. Using default.\n", key, valueString)
		} else {
			valueInt = i
		}
	}

	return valueInt
}
