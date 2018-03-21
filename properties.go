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
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Provides a series of methods and constants that revolve around the getting
// (or programmatically setting/overriding) environmental properties, returning
// default values if not set.

const (
	// EnvVarClusterName is the name of the environment variable the defines
	// the name of the cluster. Multicast messages from differently-named
	// instances are ignored.
	EnvVarClusterName = "SMUDGE_CLUSTER_NAME"

	// DefaultClusterName is the default name of the cluster for the purposes
	// of multicast announcements: multicast messages from differently-named
	// instances are ignored.
	DefaultClusterName string = "smudge"

	// EnvVarHeartbeatMillis is the name of the environment variable that
	// sets the heartbeat frequency (in millis).
	EnvVarHeartbeatMillis = "SMUDGE_HEARTBEAT_MILLIS"

	// DefaultHeartbeatMillis is the default heartbeat frequency (in millis).
	DefaultHeartbeatMillis int = 500

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

	// EnvVarListenIP is the name of the environment variable that sets
	// the listen IP.
	EnvVarListenIP = "SMUDGE_LISTEN_IP"

	// DefaultListenIP is the default listen IP.
	DefaultListenIP = "127.0.0.1"

	// EnvVarMaxBroadcastBytes is the name of the environment variable that
	// the maximum byte length for broadcast payloads. Note that increasing
	// this runs the risk of packet fragmentation and dropped messages.
	EnvVarMaxBroadcastBytes = "SMUDGE_MAX_BROADCAST_BYTES"

	// DefaultMaxBroadcastBytes is the default maximum byte length for
	// broadcast payloads. This is guided by the maximum safe UDP packet size
	// of 508 bytes, which must also contain status updates and additional
	// message overhead.
	DefaultMaxBroadcastBytes int = 256

	// EnvVarMulticastAddress is the name of the environment variable that
	// defines the multicast address that will be used.
	EnvVarMulticastAddress = "SMUDGE_MULTICAST_ADDRESS"

	// DefaultMulticastAddress is the default multicast address. Empty string
	// indicates 224.0.0.0 for IPv4 and [ff02::1] for IPv6.
	DefaultMulticastAddress string = ""

	// EnvVarMulticastEnabled is the name of the environment variable that
	// describes whether Smudge will attempt to announce its presence via
	// multicast on startup.
	EnvVarMulticastEnabled = "SMUDGE_MULTICAST_ENABLED"

	// DefaultMulticastEnabled is the default value for whether Smudge will
	// attempt to announce its presence via multicast on startup.
	DefaultMulticastEnabled string = "true"

	// EnvVarMulticastAnnounceIntervalSeconds is the name of the environment
	// variable that describes whether Smudge will attempt to re-announce its
	// presence via multicast every X seconds.
	EnvVarMulticastAnnounceIntervalSeconds = "SMUDGE_MULTICAST_ANNOUNCE_INTERVAL"

	// DefaultMulticastAnnounceIntervalSeconds is the default value for whether
	// Smudge will re-announce its presence via multicast
	DefaultMulticastAnnounceIntervalSeconds = 0

	// EnvVarMulticastPort is the name of the environment variable that
	// defines the multicast announcement listening port.
	EnvVarMulticastPort = "SMUDGE_MULTICAST_PORT"

	// DefaultMulticastPort is the default value for the multicast
	// listening port.
	DefaultMulticastPort int = 9998

	// EnvVarPingHistoryFrontload is the name of the environment variable that
	// defines the value (in milliseconds) used to pre-populate the ping
	// history buffer, which is used to dynamically calculate ping timeouts and
	// is gradually overwritten with real data over time.
	EnvVarPingHistoryFrontload = "SMUDGE_PING_HISTORY_FRONTLOAD"

	// DefaultPingHistoryFrontload is the default value (in milliseconds) used
	// to pre-populate the ping history buffer, which is used to dynamically
	// calculate ping timeouts and is gradually overwritten with real data
	// over time.
	DefaultPingHistoryFrontload = 200

	// EnvVarMinPingTime is the name of the environment variable that
	// defines the lower bound on recorded ping response times (in
	// milliseconds). This prevents the system instability and flapping that
	// can come from consistently small values.
	EnvVarMinPingTime = "SMUDGE_MIN_PING_TIME"

	// DefaultMinPingTime is default lower bound on recorded ping response
	// times (in milliseconds). This prevents the system instability and
	// flapping that can come from consistently small values.
	DefaultMinPingTime = 150
)

var clusterName string

var heartbeatMillis int

var listenPort int

var listenIP net.IP

var initialHosts []string

var maxBroadcastBytes int

var minPingTime int

var multicastEnabledString string

var multicastEnabled = true

var multicastAnnounceIntervalSeconds = 10

var multicastPort int

var multicastAddress string

var pingHistoryFrontload int

const stringListDelimitRegex = "\\s*((,\\s*)|(\\s+))"

// GetClusterName gets the name of the cluster for the purposes of
// multicast announcements: multicast messages from differently-named
// instances are ignored.
func GetClusterName() string {
	if clusterName == "" {
		clusterName = getStringVar(EnvVarClusterName, DefaultClusterName)
	}

	return clusterName
}

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

// GetListenIP returns the IP that this host will listen on.
func GetListenIP() net.IP {
	if listenIP == nil {
		listenIP = net.ParseIP(getStringVar(EnvVarListenIP, DefaultListenIP))
	}

	return listenIP
}

// GetMaxBroadcastBytes returns the maximum byte length for broadcast payloads.
func GetMaxBroadcastBytes() int {
	if maxBroadcastBytes == 0 {
		maxBroadcastBytes = getIntVar(EnvVarMaxBroadcastBytes, DefaultMaxBroadcastBytes)
	}

	return maxBroadcastBytes
}

// GetMinPingTime returns the minimum ping response time in milliseconds. Ping
// response times below this value are recorded as this minimum.
func GetMinPingTime() int {
	if minPingTime == 0 {
		minPingTime = getIntVar(EnvVarMinPingTime, DefaultMinPingTime)
	}

	return minPingTime
}

// GetMulticastEnabled returns whether multicast announcements are enabled.
func GetMulticastEnabled() bool {
	if multicastEnabledString == "" {
		multicastEnabledString = strings.ToLower(getStringVar(EnvVarMulticastEnabled, DefaultMulticastEnabled))
		multicastEnabled = len(multicastEnabledString) > 0 && []rune(multicastEnabledString)[0] == 't'
	}

	return multicastEnabled
}

// GetMulticastAnnounceIntervalSeconds returns the amount of seconds to wait between
// multicast announcements.
func GetMulticastAnnounceIntervalSeconds() int {
	multicastAnnounceIntervalSeconds = getIntVar(EnvVarMulticastAnnounceIntervalSeconds, DefaultMulticastAnnounceIntervalSeconds)
	return multicastAnnounceIntervalSeconds
}

// GetMulticastAddress returns the address the will be used for multicast
// announcements.
func GetMulticastAddress() string {
	if multicastAddress == "" {
		multicastAddress = getStringVar(EnvVarMulticastAddress, DefaultMulticastAddress)
	}

	return multicastAddress
}

// GetMulticastPort returns the defined multicast announcement listening port.
func GetMulticastPort() int {
	if multicastPort == 0 {
		multicastPort = getIntVar(EnvVarMulticastPort, DefaultMulticastPort)
	}

	return multicastPort
}

// GetPingHistoryFrontload returns the value (in milliseconds) used to
// pre-populate the ping history buffer, which is used to dynamically calculate
// ping timeouts and is gradually overwritten with real data over time.
func GetPingHistoryFrontload() int {
	if pingHistoryFrontload == 0 {
		pingHistoryFrontload = getIntVar(EnvVarPingHistoryFrontload, DefaultPingHistoryFrontload)
	}

	return pingHistoryFrontload
}

// SetClusterName sets the name of the cluster for the purposes of multicast
// announcements: multicast messages from differently-named instances are
// ignored.
func SetClusterName(val string) {
	if val == "" {
		clusterName = DefaultClusterName
	} else {
		clusterName = val
	}
}

// SetHeartbeatMillis sets this nodes heartbeat frequency. Unlike
// SetListenPort(), calling this function after Begin() has been called will
// have an effect.
func SetHeartbeatMillis(val int) {
	if val == 0 {
		heartbeatMillis = DefaultHeartbeatMillis
	} else {
		heartbeatMillis = val
	}
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

// SetListenIP sets the IP to listen on. It has no effect once
// Begin() has been called.
func SetListenIP(val net.IP) {
	if len(AllNodes()) > 0 {
		logWarn("Do not call SetListenIP() after nodes have been added, it may cause unexpected behavior.")
	}

	if val == nil {
		listenIP = net.ParseIP(DefaultListenIP)
	} else {
		listenIP = val
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

// SetMinPingTime sets the minimum ping response time in milliseconds. Ping
// response times below this value are recorded as this minimum.
func SetMinPingTime(val int) {
	if val == 0 {
		minPingTime = DefaultMinPingTime
	} else {
		minPingTime = val
	}
}

// SetMulticastAddress sets the address that will be used for multicast
// announcements.
func SetMulticastAddress(val string) {
	if val == "" {
		multicastAddress = DefaultMulticastAddress
	} else {
		multicastAddress = val
	}
}

// SetMulticastEnabled sets whether multicast announcements are enabled.
func SetMulticastEnabled(val bool) {
	multicastEnabledString = fmt.Sprintf("%v", val)
}

// SetMulticastAnnounceIntervalSeconds sets the number of seconds between multicast announcements
func SetMulticastAnnounceIntervalSeconds(val int) {
	multicastAnnounceIntervalSeconds = val
}

// SetMulticastPort sets multicast announcement listening port.
func SetMulticastPort(val int) {
	if val == 0 {
		multicastPort = DefaultMulticastPort
	} else {
		multicastPort = val
	}
}

// SetPingHistoryFrontload sets the value (in milliseconds) used to
// pre-populate the ping history buffer, which is used to dynamically calculate
// ping timeouts and is gradually overwritten with real data over time.
// Setting this to 0 will restore the default value.
func SetPingHistoryFrontload(val int) {
	if val == 0 {
		pingHistoryFrontload = DefaultPingHistoryFrontload
	} else {
		pingHistoryFrontload = val
	}
}

// Gets an environmental variable "key". If it does not exist, "defaultVal" is
// returned; if it does, it attempts to convert to an integer, returning
// "defaultVal" if it fails.
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
// "defaultVal" if it fails.
func getStringArrayVar(key string, defaultVal string) []string {
	valueString := os.Getenv(key)

	if valueString == "" {
		valueString = defaultVal
	}

	valueSlice := splitDelimmitedString(valueString, stringListDelimitRegex)

	return valueSlice
}

// Gets an environmental variable "key". If it does not exist, "defaultVal" is
// returned; if it does, it attempts to convert to a string, returning
// "defaultVal" if it fails.
func getStringVar(key string, defaultVal string) string {
	valueString := os.Getenv(key)

	if valueString == "" {
		valueString = defaultVal
	}

	return valueString
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
