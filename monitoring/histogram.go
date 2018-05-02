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
	"sync"
)

// Histogram is a simple histogram with fixed intervals. The intervals are
// exponents of 2, ie 1, 2, 4, 8... up to 2^32
type Histogram struct {
	values []int
	mutex  *sync.Mutex
}

// HistogramSize is the number of elements in the histogram.
const HistogramSize = 32

// NewHistogram creates a new Histogram instance
func NewHistogram() *Histogram {
	return &Histogram{make([]int, HistogramSize), &sync.Mutex{}}
}

// Add adds a new item to the histogram.
func (h *Histogram) Add(v float64) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	max := 1.0
	for i := 0; i < HistogramSize-1; i++ {
		if v <= max {
			h.values[i]++
			return
		}
		max *= 2
	}
	h.values[HistogramSize-1]++
}

// Values return a copy of the current logged values. The first element will
// be the values that are <= 1, the next element values that are <= 2
func (h *Histogram) Values() []int {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	var ret []int
	ret = append(ret, h.values...)
	return ret
}

// String returns a JSON representation of the histogram. The first index is
// the count of values <= 1, the next values <= 2 and so on.
func (h *Histogram) String() string {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	buf, err := json.Marshal(h.values)
	if err != nil {
		return "{}"
	}
	return string(buf)
}
