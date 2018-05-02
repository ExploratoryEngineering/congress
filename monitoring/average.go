package monitoring

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
	"encoding/json"
	"math"
	"sync"
)

// AverageGauge is a gauge with min/max/average for a fixed set of samples
type AverageGauge struct {
	samples []float64
	count   int
	index   int
	mutex   *sync.Mutex
}

// NewAverageGauge creates a new AgerageGauge instance with the given set
// of samples.
func NewAverageGauge(count int) *AverageGauge {
	return &AverageGauge{make([]float64, count), 0, 0, &sync.Mutex{}}
}

// Add adds a new value to the gauge
func (a *AverageGauge) Add(value float64) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.samples[a.index] = value
	a.index = (a.index + 1) % len(a.samples)
	a.count++
}

// Averages is the calculated averages
type Averages struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
}

// Calculate average, min and max for the current set of samples
func (a *AverageGauge) Calculate() Averages {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	ret := Averages{}
	sampleCount := len(a.samples)
	ret.Count = a.count
	if a.count < sampleCount {
		sampleCount = a.count
	}

	total := 0.0
	if sampleCount == 0 {
		return ret
	}

	ret.Max = -math.MaxFloat64
	ret.Min = math.MaxFloat64
	for i := 0; i < sampleCount; i++ {
		sample := a.samples[i]
		total += sample
		if ret.Max < sample {
			ret.Max = sample
		}
		if ret.Min > sample {
			ret.Min = sample
		}
	}
	ret.Average = total / float64(sampleCount)
	return ret
}

// String returns the values averaged and encoded as JSON
func (a *AverageGauge) String() string {
	buf, err := json.Marshal(a.Calculate())
	if err != nil {
		return "{}"
	}
	return string(buf)
}
