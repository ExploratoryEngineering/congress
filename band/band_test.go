package band

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

func TestBandFactory(t *testing.T) {
	eu, err := NewBand(EU868Band)
	if err != nil {
		t.Error(err)
	}
	if eu.Name() != "EU 863-870MHz ISM Band" {
		t.Errorf("Unexpected band name : %s", eu.Name())
	}
	us, err2 := NewBand(US915Band)
	if err2 != nil {
		t.Error(err)
	}
	if us.Name() != "US 902-928MHz ISM Band" {
		t.Errorf("Unexpected band name : %s", us.Name())
	}
	_, err3 := NewBand(CN780Band)
	if err3 == nil {
		t.Error("Did not expect this band to be implemented.")
	}
}

func TestFOptsLookup(t *testing.T) {
	mp := MaximumPayloadSize{N: 1, M: 8}

	if mp.WithFOpts() != 1 {
		t.Error("M != 1")
	}
	if mp.WithoutFOpts() != 8 {
		t.Error("N != 8")
	}

}
