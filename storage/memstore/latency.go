package memstore

//
//Copyright 2018 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import (
	"math/rand"
	"time"
)

//
// This is a simple latency emulator. It will inject some latency in calls to
// the memory database. Ordinary backend storages introduces latencies to a
// certain degree, typically 1 ms but they can go up to multiple milliseconds,
// even 40-50 in the worst case. Since the memory-backed database is quite fast
// (worst case response times are measured in microseconds) some artificial
// latency is needed to get something approximating real-world results.
// The RandomDelay function just waits for a random interval somewhere in the
// [min,max> range before returning.
//
type latencyStorage struct {
	minLatencyMs int // Minimum latency
	maxLatencyMs int // Maximum latency
}

func (m *latencyStorage) RandomDelay() {
	if m.maxLatencyMs <= 0 {
		return
	}
	latency := m.minLatencyMs*1000 + rand.Intn((m.maxLatencyMs-m.minLatencyMs)*1000)
	time.Sleep(time.Duration(latency) * time.Microsecond)
}
