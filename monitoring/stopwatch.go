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
)

// Stopwatch times an operation and updates the specified counter with the time
// taken. Time is logged in microseconds
func Stopwatch(counter *histogramCounter, opToMeasure func()) {
	start := time.Now()
	opToMeasure()
	end := time.Now()
	counter.Add(float64(end.Sub(start).Nanoseconds()) / 1000.0)
}
