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
import "testing"

func TestAverageGauge(t *testing.T) {
	ag := NewAverageGauge(10)
	for i := 0; i < 20; i++ {
		ag.Add(1.0)
	}
	ag.Add(0.5)
	ag.Add(1.5)
	av := ag.Calculate()
	if av.Average != 1.0 {
		t.Fatalf("Average isn't calculated correctly. Got %f", av.Average)
	}
	if av.Count != 22 {
		t.Fatalf("Missing counts. Got %d", av.Count)
	}
	if av.Min != 0.5 {
		t.Fatalf("Min is wrong. Got %f", av.Min)
	}
	if av.Max != 1.5 {
		t.Fatalf("Max is wrong. Got %f", av.Max)
	}
}

func TestSingleSample(t *testing.T) {
	ag := NewAverageGauge(10)
	ag.Add(1000.0)

	av := ag.Calculate()
	right := Averages{Average: 1000.0, Count: 1, Min: 1000.0, Max: 1000.0}
	if av != right {
		t.Fatalf("Single sample failed: result = %+v", av)
	}
}

func TestNoSamples(t *testing.T) {
	ag := NewAverageGauge(10)
	av := ag.Calculate()
	right := Averages{Average: 0.0, Count: 0, Min: 0.0, Max: 0.0}
	if av != right {
		t.Fatalf("Zero sample failed: result = %+v", av)
	}
}
