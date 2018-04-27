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
	"time"

	"github.com/ExploratoryEngineering/logging"
)

// Timer is responsible for timings throughout the pipeline. Timing
// starts when Begin() is called and is completed when End() is called.
type Timer struct {
	section *histogramCounter
	start   time.Time
}

// Begin starts timing for a new section. The provided counter is updated with
// the elapsed time when there's a call to End. Begin and End should be called
// after one another. If begin is called more than once an error will be logged.
func (t *Timer) Begin(section *histogramCounter) {
	if section == nil {
		logging.Error("Section can't be nil")
		return
	}
	if t.section != nil {
		logging.Error("Section is already being timed (is %v, new is %v). Ignoring Begin", t.section.name, section.name)
		return
	}
	t.start = time.Now()
	t.section = section
}

// End stops timing for a section. Begin must be called before End is called.
// If there's no timing running at the moment it will log an error and return.
// The elapsed time from Begin to End will be logged in the counter provided
// to the Begin call. Time is logged in microseconds.
func (t *Timer) End() {
	if t.section == nil {
		logging.Error("No corresponding Begin() for End(). Ignoring End")
		return
	}
	duration := time.Now().Sub(t.start)
	t.section.Add(float64(duration.Nanoseconds()) / 1000.0)
	t.section = nil
}

// NewTimer creates a new timer
func NewTimer() Timer {
	return Timer{nil, time.Now()}
}
