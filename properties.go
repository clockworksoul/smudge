package blackfish

import (
	"fmt"
	"os"
	"strconv"
)

// Provides a series of methods and constants that revolve around the getting
// (or programmatically setting/overriding) environmental properties, returning
// default values if not set.

const (
	// The default heartbeat frequency in milliseconds for each node.
	ENV_HEARTBEAT_MILLIS         = "BLACKFISH_HEARTBEAT_MILLIS"
	DEFAULT_HEARTBEAT_MILLIS int = 500

	// The port to listen on by default.
	ENV_LISTEN_PORT         = "BLACKFISH_LISTEN_PORT"
	DEFAULT_LISTEN_PORT int = 9999
)

var heartbeat_millis int

var listen_port int

// The port to listen on.
func GetListenPort() int {
	if listen_port == 0 {
		listen_port = getIntVar(ENV_LISTEN_PORT, DEFAULT_LISTEN_PORT)
	}

	return listen_port
}

// The port to listen on.
func SetListenPort(p int) {
	if p == 0 {
		listen_port = DEFAULT_LISTEN_PORT
	} else {
		listen_port = p
	}
}

// The heartbeat frequency in milliseconds for each node.
func GetHeartbeatMillis() int {
	if heartbeat_millis == 0 {
		heartbeat_millis = getIntVar(ENV_HEARTBEAT_MILLIS, DEFAULT_HEARTBEAT_MILLIS)
	}

	return heartbeat_millis
}

// The heartbeat frequency in milliseconds for each node.
func SetHeartbeatMillis(val int) {
	heartbeat_millis = val
}

// Gets an environmental variable "key". If it does not exist, "defval" is
// returned; if it does, it attempts to convert to an integer, returning
// "defval" is it fails.
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
