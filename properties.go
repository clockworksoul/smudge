package blackfish

import (
	"os"
	"strconv"
)

// Provides a series of methods and constants that revolve around the getting
// (or programmatically setting/overriding) environmental properties, returning
// default values if not set.

const (
	// The default heartbeat frequency in milliseconds for each node.
	EnvVarHeartbeatMillis      = "BLACKFISH_HEARTBEAT_MILLIS"
	DefaultHeartbeatMillis int = 500

	// The port to listen on by default.
	EnvVarListenPort      = "BLACKFISH_LISTEN_PORT"
	DefaultListenPort int = 9999
)

var heartbeatMillis int

var listenPort int

// The port to listen on.
func GetListenPort() int {
	if listenPort == 0 {
		listenPort = getIntVar(EnvVarListenPort, DefaultListenPort)
	}

	return listenPort
}

// The port to listen on.
func SetListenPort(p int) {
	if p == 0 {
		listenPort = DefaultListenPort
	} else {
		listenPort = p
	}
}

// The heartbeat frequency in milliseconds for each node.
func GetHeartbeatMillis() int {
	if heartbeatMillis == 0 {
		heartbeatMillis = getIntVar(EnvVarHeartbeatMillis, DefaultHeartbeatMillis)
	}

	return heartbeatMillis
}

// The heartbeat frequency in milliseconds for each node.
func SetHeartbeatMillis(val int) {
	heartbeatMillis = val
}

// Gets an environmental variable "key". If it does not exist, "defaultVal" is
// returned; if it does, it attempts to convert to an integer, returning
// "defaultVal" is it fails.
func getIntVar(key string, defaultVal int) int {
	valueString := os.Getenv(key)
	valueInt := defaultVal

	if valueString != "" {
		i, err := strconv.Atoi(key)

		if err != nil {
			logfWarn("Failed to parse env property %s: %s is not "+
				"an integer. Using default.\n", key, valueString)
		} else {
			valueInt = i
		}
	}

	return valueInt
}
