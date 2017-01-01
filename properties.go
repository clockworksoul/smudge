/*
Copyright 2016 The Smudge Authors.

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

package smudge

import (
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Provides a series of methods and constants that revolve around the getting
// (or programmatically setting/overriding) environmental properties, returning
// default values if not set.

const (
	// EnvVarHeartbeatMillis is the name of the environment variable that
	// sets the heartbeat frequency (in millis).
	EnvVarHeartbeatMillis = "SMUDGE_HEARTBEAT_MILLIS"

	// DefaultHeartbeatMillis is the default heartbeat frequency (in millis).
	DefaultHeartbeatMillis int = 250

	// EnvVarInitialHosts is the name of the environment variable that sets
	// the initial known hosts. The value it sets should be a comma-delimitted
	// string of one or more IP:PORT pairs (port is optional if it matched the
	// value of SMUDGE_LISTEN_PORT).
	EnvVarInitialHosts = "SMUDGE_INITIAL_HOSTS"

	// DefaultInitialHosts default lists of initially known hosts.
	DefaultInitialHosts string = ""

	// EnvVarListenPort is the name of the environment variable that sets
	// the UDP listen port.
	EnvVarListenPort = "SMUDGE_LISTEN_PORT"

	// DefaultListenPort is the default UDP listen port.
	DefaultListenPort int = 9999

	// EnvVarMaxBroadcastBytes is the name of the environment variable that
	// the maximum byte length for broadcast payloads. Note that increasing
	// this runs the risk of packet fragmentation and dropped messages.
	EnvVarMaxBroadcastBytes = "SMUDGE_MAX_BROADCAST_BYTES"

	// DefaultMaxBroadcastBytes is the default maximum byte length for
	// broadcast payloads. This is guided by the maximum safe UDP packet size
	// of 508 bytes, which must also contain status updates and additional
	// message overhead.
	DefaultMaxBroadcastBytes int = 256
)

var heartbeatMillis int

var listenPort int

var initialHosts []string

var maxBroadcastBytes int

const stringListDelimitRegex = "\\s*,?\\s+"

// GetHeartbeatMillis gets this host's heartbeat frequency in milliseconds.
func GetHeartbeatMillis() int {
	if heartbeatMillis == 0 {
		heartbeatMillis = getIntVar(EnvVarHeartbeatMillis, DefaultHeartbeatMillis)
	}

	return heartbeatMillis
}

// GetInitialHosts returns the list of initially known hosts.
func GetInitialHosts() []string {
	if initialHosts == nil {
		initialHosts = getStringArrayVar(EnvVarInitialHosts, DefaultInitialHosts)
	}

	return initialHosts
}

// GetListenPort returns the port that this host will listen on.
func GetListenPort() int {
	if listenPort == 0 {
		listenPort = getIntVar(EnvVarListenPort, DefaultListenPort)
	}

	return listenPort
}

// GetMaxBroadcastBytes returns the maximum byte length for broadcast payloads.
func GetMaxBroadcastBytes() int {
	if maxBroadcastBytes == 0 {
		maxBroadcastBytes = getIntVar(EnvVarMaxBroadcastBytes, DefaultMaxBroadcastBytes)
	}

	return maxBroadcastBytes
}

// SetHeartbeatMillis sets this nodes heartbeat frequency. Unlike
// SetListenPort(), calling this function after Begin() has been called will
// have an effect.
func SetHeartbeatMillis(val int) {
	if val == 0 {
		heartbeatMillis = DefaultListenPort
	} else {
		heartbeatMillis = val
	}

	heartbeatMillis = val
}

// SetListenPort sets the UDP port to listen on. It has no effect once
// Begin() has been called.
func SetListenPort(val int) {
	if val == 0 {
		listenPort = DefaultListenPort
	} else {
		listenPort = val
	}
}

// SetMaxBroadcastBytes sets the maximum byte length for broadcast payloads.
// Note that increasing this beyond the default of 256 runs the risk of packet
// fragmentation and dropped messages.
func SetMaxBroadcastBytes(val int) {
	if val == 0 {
		maxBroadcastBytes = DefaultMaxBroadcastBytes
	} else {
		maxBroadcastBytes = val
	}
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

// Gets an environmental variable "key". If it does not exist, "defaultVal" is
// returned; if it does, it attempts to convert to a string slice, returning
// "defaultVal" is it fails.
func getStringArrayVar(key string, defaultVal string) []string {
	valueString := os.Getenv(key)

	if valueString == "" {
		valueString = defaultVal
	}

	valueSlice := splitDelimmitedString(valueString, stringListDelimitRegex)

	return valueSlice
}

// Splits a string on a regular expression.
func splitDelimmitedString(str string, regex string) []string {
	var result []string

	str = strings.TrimSpace(str)

	if str != "" {
		reg := regexp.MustCompile(regex)
		indices := reg.FindAllStringIndex(str, -1)

		result = make([]string, len(indices)+1)

		lastStart := 0
		for i, val := range indices {
			result[i] = str[lastStart:val[0]]
			lastStart = val[1]
		}

		result[len(indices)] = str[lastStart:]

		// Special case of single empty string
		if len(result) == 1 && result[0] == "" {
			result = make([]string, 0, 0)
		}
	}

	return result
}
