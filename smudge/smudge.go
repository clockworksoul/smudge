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

package main

import (
	"flag"
	"fmt"
	"github.com/clockworksoul/smudge"
	"github.com/rs/zerolog/log"
)

type logger struct {
}

// Log writes a log message of a certain level to the logger
func (l logger) Log(level smudge.LogLevel, a ...interface{}) (n int, err error) {
	str := fmt.Sprint(a...)
	switch level {
	case smudge.LogFatal:
		log.Fatal().Msg(str)
	case smudge.LogInfo:
		log.Info().Msg(str)
	case smudge.LogError:
		log.Error().Msg(str)
	case smudge.LogWarn:
		log.Warn().Msg(str)
	default:
		log.Debug().Msg(str)
	}
	return 0, nil
}

// Log writes a log message of a certain level to the logger
func (l logger) Logf(level smudge.LogLevel, format string, a ...interface{}) (n int, err error) {
	switch level {
	case smudge.LogFatal:
		log.Fatal().Msgf(format, a...)
	case smudge.LogInfo:
		log.Info().Msgf(format, a...)
	case smudge.LogError:
		log.Error().Msgf(format, a...)
	case smudge.LogWarn:
		log.Warn().Msgf(format, a...)
	default:
		log.Debug().Msgf(format, a...)
	}
	return 0, nil
}

func main() {
	var nodeAddress string
	var heartbeatMillis int
	var listenPort int
	var err error

	flag.StringVar(&nodeAddress, "node", "", "Initial node")

	flag.IntVar(&listenPort, "port",
		int(smudge.GetListenPort()),
		"The bind port")

	flag.IntVar(&heartbeatMillis, "hbf",
		int(smudge.GetHeartbeatMillis()),
		"The heartbeat frequency in milliseconds")

	flag.Parse()

	ip, err := smudge.GetLocalIP()
	if err != nil {
		log.Fatal().Msgf("Could not get local ip: %s\n", err)
	}

	smudge.SetLogThreshold(smudge.LogInfo)
	smudge.SetLogger(logger{})
	smudge.SetListenPort(listenPort)
	smudge.SetHeartbeatMillis(heartbeatMillis)
	smudge.SetListenIP(ip)

	if ip.To4() == nil {
		smudge.SetMaxBroadcastBytes(512) // 512 for IPv6
	}

	if nodeAddress != "" {
		node, err := smudge.CreateNodeByAddress(nodeAddress)

		if err == nil {
			smudge.AddNode(node)
		} else {
			fmt.Println(err)
		}
	}

	if err == nil {
		smudge.Begin()
	} else {
		fmt.Println(err)
	}
}
