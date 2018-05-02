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
	"testing"
	"time"
)

func testTotalCount(t *testing.T, ts *TimeSeries, expected uint32) {
}

func TestEmpty(t *testing.T) {
	x := NewTimeSeries(Hours)
	x.GetCounts()
}

// Simple test that increments 1000 times. Time *might* be in the same minute
func TestSimpleIncrement(t *testing.T) {
	const incrementCount = 1000
	ts := NewTimeSeries(Minutes)
	for i := 0; i < incrementCount; i++ {
		ts.Increment()
	}
	testTotalCount(t, ts, incrementCount)
}

// generic increment-and-skip test for a time series
func testWithSkip(t *testing.T, ts *TimeSeries, skip time.Duration, count int, expected int) {
	currentTime := time.Date(2017, 1, 1, 0, 0, 0, 0, time.Local)
	ct := &currentTime
	ts.timeKeeper = func() time.Time {
		return *ct
	}
	ts.lastTime = ts.timeKeeper()

	for i := 0; i < count; i++ {
		ts.Increment()
		currentTime = currentTime.Add(skip)
	}
	// Step back 1 skip to compensate for the last
	currentTime = currentTime.Add(-skip)
	total := uint32(0)
	for _, v := range ts.GetCounts() {
		total += v
	}
	// If we sum all of the entries it should be exactly the count
	// we incremented
	if total != uint32(expected) {
		t.Fatalf("Expected %d increments in minutes but found %d (return is %v)", expected, total, ts.GetCounts())
	}
}

func TestMinutesSkipSecond(t *testing.T) {
	ts := NewTimeSeries(Minutes)
	testWithSkip(t, ts, 1*time.Second, 60*60, 60*60)
	for i, v := range ts.GetCounts() {
		if v != 60 {
			t.Fatalf("Expected 60 items on index %d not %d", i, v)
		}
	}
}

// Count with minute resolution, skip one minute for each
// iteration
func TestMinutesSkipMinute(t *testing.T) {
	ts := NewTimeSeries(Minutes)
	testWithSkip(t, ts, 1*time.Minute, 60, 60)
	// Each minute should contain 1 increment
	for i, v := range ts.GetCounts() {
		if v != 1 {
			t.Fatalf("Expected 1 items on index %d not %d", i, v)
		}
	}
}

// Test minute intervals with increments of one hour
func TestMinutesSkipHour(t *testing.T) {
	ts := NewTimeSeries(Minutes)
	testWithSkip(t, ts, 1*time.Hour, 24, 1)
	// the first minute should contain 1 increment
	if ts.GetCounts()[int(Minutes)-1] != 1 {
		t.Fatalf("Expected last increment to be 1 but it was %d (v=%v)", ts.GetCounts()[0], ts.GetCounts())
	}
}

// Test hourly resolution but skip one minute at a time
func TestHoursSkipMinute(t *testing.T) {
	ts := NewTimeSeries(Hours)
	testWithSkip(t, ts, 1*time.Minute, 60*24, 60*24)
	// Each hour should contain 60 increments
	for i, v := range ts.GetCounts() {
		if v != 60 {
			t.Fatalf("Expected 60 items on index %d not %d", i, v)
		}
	}
}

func TestDaysSkipHours(t *testing.T) {
	ts := NewTimeSeries(Days)
	// Do two iterations but only the last 30 days are stored
	testWithSkip(t, ts, 1*time.Hour, 24*60, 24*30)
	for i, v := range ts.GetCounts() {
		if v != 24 {
			t.Fatalf("Expected 24 items on index %d not %d (%v)", i, v, ts.counts)
		}
	}
}

func BenchmarkTimeSeries(b *testing.B) {
	ts := NewTimeSeries(Minutes)
	for i := 0; i < b.N; i++ {
		ts.Increment()
	}
}

func TestTimeSeriesSingle(t *testing.T) {
	ts := NewTimeSeries(Minutes)
	ts.Increment()
	counts := ts.GetCounts()
	if counts[int(Minutes)-1] != 1 {
		t.Fatalf("Last item should be 1 (returned = %v)", counts)
	}
}

func TestTimeSeriesDouble(t *testing.T) {
	ts := NewTimeSeries(Minutes)
	currentTime := time.Date(2017, 1, 1, 0, 0, 0, 0, time.Local)
	ct := &currentTime
	ts.timeKeeper = func() time.Time {
		return *ct
	}
	ts.lastTime = ts.timeKeeper()

	ts.Increment()
	currentTime = currentTime.Add(59 * time.Minute)
	ts.Increment()
	ts.Increment()

	counts := ts.GetCounts()
	if counts[0] != 1 {
		t.Fatalf("59 minutes ago isn't 1 (v = %v)", counts)
	}
	if counts[int(Minutes)-1] != 2 {
		t.Fatalf("now isn't 2 (v = %v)", counts)
	}
}
func TestTimeSeries(t *testing.T) {
	ts := NewTimeSeries(Minutes)

	currentTime := time.Date(2017, 1, 1, 0, 0, 0, 0, time.Local)
	ct := &currentTime
	ts.timeKeeper = func() time.Time {
		return *ct
	}
	ts.lastTime = ts.timeKeeper()
	// Increment the time series 0..59 times for each minute
	for i := 0; i < 60; i++ {
		currentTime = currentTime.Add(60 * time.Second)
		n := 0
		for j := 0; j < i; j++ {
			ts.Increment()
			n++
		}
	}

	counts := ts.GetCounts()
	if len(counts) != int(Minutes) {
		t.Fatalf("Expected returned array to be %d elements but it is %d", Minutes, len(counts))
	}
	// The first element should be 0, the next 1 and so on
	for i, v := range counts {
		if v != uint32(i) {
			t.Fatalf("Expected value %d at index %d but got %d (array is %v)", i, i, v, counts)
		}
	}

	// Skip forward an hour. The counts should be all 0
	currentTime = currentTime.Add(60 * time.Minute)
	counts = ts.GetCounts()

	// The first element should be 0, the next 1 and so on
	for i, v := range counts {
		if v != 0 {
			t.Fatalf("Expected value 0 at index %d but got %d (array is %v)", i, v, counts)
		}
	}
}
