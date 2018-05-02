package protocol

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

func TestFCtrlEncodeDecode(t *testing.T) {
	f1 := FCtrl{
		ADR:       true,
		ADRACKReq: false,
		ACK:       true,
		FPending:  false,
		ClassB:    false,
		FOptsLen:  15,
	}

	buffer := make([]byte, 1024)
	pos := 0
	if err := f1.encode(buffer, &pos); err != nil {
		t.Fatalf("Got error encoding FCtrl: %v", err)
	}
	f2 := FCtrl{}

	pos = 0
	if err := f2.decode(buffer, &pos); err != nil {
		t.Fatalf("Got error decoding FCtrl: %v", err)
	}

	if f1 != f2 {
		t.Fatalf("Decoded and encoded are not the same: %+v != %+v", f1, f2)
	}
}

func TestFCtrlEncodeInvalidFOptsLen(t *testing.T) {
	f1 := FCtrl{
		ADR:       true,
		ADRACKReq: true,
		ACK:       true,
		FPending:  true,
		ClassB:    true,
		FOptsLen:  99,
	}

	buffer := make([]byte, 1024)
	pos := 0
	if err := f1.encode(buffer, &pos); err == nil {
		t.Fatal("Expected error when FOptsLen was too big but got none")
	}
}

func TestFCtrlBufferRangeChecks(t *testing.T) {
	basicEncoderTests(t, &FCtrl{})
	basicDecoderTests(t, &FCtrl{})
}
