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

func TestHistogram(t *testing.T) {

	h := NewHistogram()

	for j := 0; j < 1000; j++ {
		index := 1.0
		for i := 0; i < HistogramSize; i++ {
			h.Add(index - 0.1)
			index *= 2
		}
	}

	for i, v := range h.Values() {
		if v != 1000 {
			t.Fatalf("Did not get the expected count for index %d: %v (result=%+v)", i, v, h.Values())
		}
	}

	h.Add(1000)
	// This would be in the 2^10th bucket: (1,2,4,8,16,32,64,128,256,512,1024)
	if h.Values()[10] != 1001 {
		t.Fatalf("Increment in the wrong place %+v", h.Values())
	}
}
