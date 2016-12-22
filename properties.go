package blackfish

import (
	"os"
	"strconv"
)

// Provides a series of methods and constants that revolve around the getting
// (or programmatically setting/overriding) environmental properties, returning
// default values if not set.

const (
	// EnvVarHeartbeatMillis is the name of the environment variable that
	// sets the heartbeat frequency (in millis).
	EnvVarHeartbeatMillis = "BLACKFISH_HEARTBEAT_MILLIS"

	// DefaultHeartbeatMillis is the default heartbeat frequency (in millis).
	DefaultHeartbeatMillis int = 500

	// EnvVarHeartbeatMillis is the name of the environment variable that sets
	// the UDP listen port.
	EnvVarListenPort = "BLACKFISH_LISTEN_PORT"

	// DefaultListenPort is the default UDP listen port.
	DefaultListenPort int = 9999
)

var heartbeatMillis int

var listenPort int

// GetListenPort returns the port that this host will listen on.
func GetListenPort() int {
	if listenPort == 0 {
		listenPort = getIntVar(EnvVarListenPort, DefaultListenPort)
	}

	return listenPort
}

// SetListenPort sets the UDP port to listen on. It has no effect once
// Begin() has been called.
func SetListenPort(p int) {
	if p == 0 {
		listenPort = DefaultListenPort
	} else {
		listenPort = p
	}
}

// GetHeartbeatMillis gets this host's heartbeat frequency in milliseconds.
func GetHeartbeatMillis() int {
	if heartbeatMillis == 0 {
		heartbeatMillis = getIntVar(EnvVarHeartbeatMillis, DefaultHeartbeatMillis)
	}

	return heartbeatMillis
}

// SetHeartbeatMillis sets this nodes heartbeat frequency. Unlike
// SetListenPort(), calling this function after Begin() has been called will
// have an effect.
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
