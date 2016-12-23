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
	"math"
	"sync"
)

type pingData struct {
	sync.RWMutex

	// The ping data. Initialized with default values by NewPingData()
	pings []uint32

	// The index in pings where the next datapoint will be added
	pointer int

	// The last calulcated mean. Recalculated if updated is true
	lastMean float64

	// The last calulcated standard deviation. Recalculated if updated is true
	lastStddev float64

	// The modified flag. Set to true when a datapoint is added
	updated bool
}

func newPingData(initialAverage int, historyCount int) pingData {
	newPings := make([]uint32, historyCount, historyCount)

	for i := 0; i < historyCount; i++ {
		newPings[i] = uint32(initialAverage)
	}

	return pingData{pings: newPings, updated: true}
}

func (pd *pingData) add(datapoint uint32) {
	pd.Lock()

	pd.pings[pd.pointer] = datapoint

	// Advance the pointer
	pd.pointer++
	pd.pointer %= len(pd.pings)

	pd.updated = true

	pd.Unlock()
}

// mean returns the simple mean (average) of the collected datapoints.
func (pd *pingData) mean() float64 {
	pd.data()

	return pd.lastMean
}

// Returns the mean modified by the requested number of sigmas
func (pd *pingData) nSigma(sigmas float64) float64 {
	mean, stddev := pd.data()

	return mean + (sigmas * stddev)
}

// stddev returns the standard deviation of the collected datapoints
func (pd *pingData) stddev() float64 {
	pd.data()

	return pd.lastStddev
}

// Returns both mean and standard deviation
func (pd *pingData) data() (float64, float64) {
	if pd.updated {
		pd.Lock()

		// Calculate the mean
		var accumulator float64
		for _, d := range pd.pings {
			accumulator += float64(d)
		}
		pd.lastMean = accumulator / float64(len(pd.pings))

		// Subtract the mean and square the result; calculcate the mean
		accumulator = 0.0 // Reusing accumulator.
		for _, d := range pd.pings {
			diff := pd.lastMean - float64(d)
			accumulator += math.Pow(diff, 2.0)
		}
		squareDiffMean := accumulator / float64(len(pd.pings))

		// Sqrt the square diffs mean and we have our stddev
		pd.lastStddev = math.Sqrt(squareDiffMean)

		pd.updated = false

		pd.Unlock()
	}

	return pd.lastMean, pd.lastStddev
}
