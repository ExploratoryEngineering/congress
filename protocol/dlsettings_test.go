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

func TestDLSettingsEncoding(t *testing.T) {
	// Use 2x 0xFF -- they should be masked into one single byte with 7 bits set
	dl := DLSettings{RX1DRoffset: 0xFF, RX2DataRate: 0xFF}
	buffer := make([]byte, 1)
	pos := 0
	if err := dl.encode(buffer, &pos); err != nil {
		t.Fatal("Got error encoding into buffer: ", err)
	}
	if buffer[0] != 0x7F {
		t.Fatalf("Expected 01111111 (0x7F) from encoding but got %02x", buffer[0])
	}

	// Repeat. Bits should now be 01110000 = 0x70
	dl.RX2DataRate = 0
	pos = 0
	if err := dl.encode(buffer, &pos); err != nil {
		t.Fatal("Got error encoding into buffer: ", err)
	}
	if buffer[0] != 0x70 {
		t.Fatalf("Expected 01110000 (0x70) from encoding but got %02x", buffer[0])
	}

	// Repeat but with pos too big.
	if err := dl.encode(buffer, &pos); err == nil {
		t.Fatal("Expected error when invalid pos is used but didn't get one")
	}
}

func TestDLSettingsEncodeDecode(t *testing.T) {
	d1 := DLSettings{RX1DRoffset: 0x7, RX2DataRate: 0xF}
	buffer := make([]byte, 2)
	pos := 0
	if err := d1.encode(buffer, &pos); err != nil {
		t.Fatalf("Got error encoding DLSettings: %v", err)
	}

	d2 := DLSettings{}
	pos = 0
	if err := d2.decode(buffer, &pos); err != nil {
		t.Fatalf("Got error decoding DLSettings: %v", err)
	}

	if d1 != d2 {
		t.Fatalf("encoded and decoded are different: %+v != %+v", d1, d2)
	}
}

func TestDLSettingsBufferRange(t *testing.T) {
	basicDecoderTests(t, &DLSettings{})
	basicEncoderTests(t, &DLSettings{})
}
