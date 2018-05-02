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
import (
	"testing"
)

func TestMACPayloadInvalidPort(t *testing.T) {
	m := NewMACPayload(ConfirmedDataUp)
	m.FPort = 0
	m.FRMPayload = []byte{1, 2, 3}

	buffer := make([]byte, 1024)
	count := 0
	if err := m.encode(buffer, &count); err != ErrParameterOutOfRange {
		t.Fatal("Expected error when port == 0 and FRMPayload is set")
	}

	m.FPort = 224
	if err := m.encode(buffer, &count); err != ErrParameterOutOfRange {
		t.Fatal("Expected error when port > 223")
	}
}

// There will be a port number if the payload contains data; if it is
// MAC commands it will be 0, if it is user data it will be 1-223
// If there's no bytes in the payload the port number will be omitted
func TestMACPayloadPortNoPort(t *testing.T) {
	m := NewMACPayload(ConfirmedDataUp)

	encode := func(m *MACPayload) []byte {
		buffer := make([]byte, 1024)
		pos := 0
		if err := m.encode(buffer, &pos); err != nil {
			t.Fatalf("Got error encoding: %v", err)
		}
		return buffer
	}

	decode := func(buf []byte, m *MACPayload) {
		pos := 0
		if err := m.encode(buf, &pos); err != nil {
			t.Fatalf("Got error encoding: %v", err)
		}
	}

	// Payload, port, and no mac commands
	m.FRMPayload = []byte{1, 2, 3, 4, 5}
	m.FPort = 1
	m.MACCommands.Clear()
	buf := encode(&m)
	decode(buf, &m)
	if m.FPort != 1 {
		t.Fatal("Port not matching")
	}
	if m.MACCommands.Size() != 0 {
		t.Fatal("Did not expect MAC commands")
	}
	// No payload, port set to 1, mac commands
	m.FRMPayload = []byte{}
	m.FPort = 2
	m.MACCommands.Add(NewUplinkMACCommand(DevStatusReq))
	buf = encode(&m)
	decode(buf, &m)
	if m.FPort != 0 {
		t.Fatal("Port should be 0")
	}
	if m.MACCommands.Size() != 1 || !m.MACCommands.Contains(DevStatusReq) {
		t.Fatal("Not expected MAC command payload")
	}

	// Port should be 0 (actually not set but 0 is as close as possible)
	m.MACCommands.Clear()
	m.FRMPayload = []byte{}
	m.FPort = 99
	buf = encode(&m)
	m.FPort = 200
	decode(buf, &m)
	if m.FPort != 0 {
		t.Fatal("Port should be 0 with no mac and no payload")
	}
	if m.MACCommands.Size() != 0 || len(m.FRMPayload) > 0 {
		t.Fatal("No payload expected")
	}
}

// Rudimentary tests on encoding and decoding
func TestMACPayloadRangeChecks(t *testing.T) {
	m1 := NewMACPayload(ConfirmedDataUp)
	m1.FPort = 123
	m1.FRMPayload = []byte{1, 2, 3}
	basicEncoderTests(t, &m1)
	basicDecoderTests(t, &m1)
}
