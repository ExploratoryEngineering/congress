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
	"time"
)

// TimeSeries represents a time series of values, grouped by a fixed interval
// (minute, hour, day). Use the Increment() value to increase the counter and
// GetCounters() to get an array of the recorded values. As a side effect
// the output will be a rate of events per time interval -- ie if you keep track
// of new items and call Increment() every time a new item is created you'll get
// the rate of new elements per time interval out from GetCounters()
//
type TimeSeries struct {
	counts  []uint32
	current int // The current index
	last    int // The last minute/hour/day used. Keeps track of wall clock time
	// This is by default time.Now() but can be changed for testing purposes
	timeKeeper func() time.Time
	lastTime   time.Time
	mutex      *sync.Mutex
	diffFunc   func(t time.Duration) int // Return time difference as minute/hour/day
	timeFunc   func(t time.Time) int     // Return minute, hour, day of time
}

// Internal type
type intervalType int

// This is the intervals that can be used for TimeSeries instances
const (
	Minutes = intervalType(60)
	Hours   = intervalType(24)
	Days    = intervalType(30)
)

// NewTimeSeries creates a new time series. The identifier
func NewTimeSeries(interval intervalType) *TimeSeries {
	ret := &TimeSeries{
		counts:     make([]uint32, interval),
		timeKeeper: time.Now,
		mutex:      &sync.Mutex{},
		lastTime:   time.Now(),
		current:    0,
		last:       0,
	}
	switch interval {
	case Minutes:
		ret.diffFunc = func(t time.Duration) int {
			return int(t.Minutes())
		}
		ret.timeFunc = func(t time.Time) int {
			return t.Minute()
		}
	case Hours:
		ret.diffFunc = func(t time.Duration) int {
			return int(t.Hours())
		}
		ret.timeFunc = func(t time.Time) int {
			return t.Hour()
		}
	case Days:
		ret.diffFunc = func(t time.Duration) int {
			return int(math.Floor(t.Hours() / 24.0))
		}
		ret.timeFunc = func(t time.Time) int {
			return t.Day() - 1
		}
	}
	return ret
}

// Move index pointer
func (t *TimeSeries) moveIndex(now time.Time) bool {
	changedPointer := false
	len := len(t.counts)
	indexDiff := t.diffFunc(now.Sub(t.lastTime))
	if indexDiff > 0 {
		// Move index forward corresponding to the time elapsed and
		// reset the counts.
		for i := t.current + 1; i < t.current+indexDiff; i++ {
			t.counts[i%len] = 0
		}
		t.current = (t.current + indexDiff) % len
		t.counts[t.current] = 0
		changedPointer = true
	}
	t.lastTime = now
	return changedPointer
}

// Increment the current minute, hour, day and month counters
func (t *TimeSeries) Increment() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	len := len(t.counts)

	now := t.timeKeeper()

	if !t.moveIndex(now) {
		time := t.timeFunc(now)
		if time != t.last {
			// The time index has changed. Move forward one step, reset old
			// counter.
			t.last = time
			t.current = (t.current + 1) % len
			t.counts[t.current] = 0
		}
	}

	t.counts[t.current]++
	t.lastTime = now
	t.last = t.timeFunc(now)
}

// GetCounts returns the counts for each time unit. The oldest item
// it at index 0, the newest item at the end of the array
func (t *TimeSeries) GetCounts() []uint32 {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Zero out the increments
	t.moveIndex(t.timeKeeper())

	l := len(t.counts)
	ret := make([]uint32, l)
	pos := 0
	for i := 0; i < l; i++ {
		ret[pos] = t.counts[(t.current+i+1)%l]
		pos++
	}
	return ret
}

// MarshalJSON dumps the time series as an array
func (t *TimeSeries) MarshalJSON() ([]byte, error) {
	val := t.GetCounts()
	return json.Marshal(&val)
}

// Convert into a JSON string. This satisifies the expvar.Var interface and
// makes it possible to expose the variable.
func (t *TimeSeries) String() string {
	buf, err := json.Marshal(t.GetCounts())
	if err != nil {
		return "{}"
	}
	return string(buf)
}
