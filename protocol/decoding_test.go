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
	"encoding/base64"
	"fmt"
	"testing"
)

// Basic decoder tests
type decoder interface {
	decode(buffer []byte, pos *int) error
}

func basicDecoderTests(t *testing.T, d decoder) {
	buffer := make([]byte, 2048)
	pos := 2049
	if err := d.decode(buffer, &pos); err == nil {
		t.Fatal("Expected error with (less than) zero length buffer")
	}
	if err := d.decode(nil, &pos); err == nil {
		t.Fatal("Expected error with nil buffer")
	}
	if err := d.decode(buffer, nil); err == nil {
		t.Fatal("Expected error with nil pointer")
	}
}

/// --- old stuff below
func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v != %v", a, b)
	}
	t.Fatal(message)
}

/*
   Known device address:   0x01E672E6
                           0000 0000 0111 0010 1110 0110 0000 0001
   Netork ID mask:         1111 1110 0000 0000 0000 0000 0000 0000
   Network address mask:   0000 0001 1111 1111 1111 1111 1111 1111
*/

func TestUnmarshalDeviceAddress_Sample1(t *testing.T) {

	data := "gOZy5gGAAQALqBJvwTWKKB0="
	sDec, _ := base64.StdEncoding.DecodeString(data)

	payload := NewPHYPayload(Proprietary)
	err := payload.UnmarshalBinary(sDec)
	if err != nil {
		t.Fatal(err)
	}

	var ExpectedNetworkID uint8
	var ExpectedNetworkAddress uint32 = 0x01E672E6

	assertEqual(t, payload.MACPayload.FHDR.DevAddr.NwkID, ExpectedNetworkID, "Incorrect network ID.")
	assertEqual(t, payload.MACPayload.FHDR.DevAddr.NwkAddr, ExpectedNetworkAddress, "Incorrect network address.")
}

func TestUnmarshalDeviceAddress_Sample2(t *testing.T) {

	data := "YOZy5gGgCABYNqzg"
	sDec, _ := base64.StdEncoding.DecodeString(data)

	payload := NewPHYPayload(Proprietary)
	err := payload.UnmarshalBinary(sDec)
	if err != nil {
		t.Fatal(err)
	}

	var ExpectedNetworkID uint8
	var ExpectedNetworkAddress uint32 = 0x01E672E6

	assertEqual(t, payload.MACPayload.FHDR.DevAddr.NwkID, ExpectedNetworkID, "Incorrect network ID.")
	assertEqual(t, payload.MACPayload.FHDR.DevAddr.NwkAddr, ExpectedNetworkAddress, "Incorrect network address.")
}

func TestUnmarshalMajorVersionIDDecoding_Sample3(t *testing.T) {

	data := "YOZy5gGgCQAyiPuX"
	sDec, _ := base64.StdEncoding.DecodeString(data)

	payload := NewPHYPayload(Proprietary)
	err := payload.UnmarshalBinary(sDec)
	if err != nil {
		t.Fatal(err)
	}

	var ExpectedMajorVersion uint8 // = 0

	assertEqual(t, payload.MHDR.MajorVersion, ExpectedMajorVersion, "Incorrect Major Version.")
}

func TestUnmarshalFOpts(t *testing.T) {
	data := "YOZy5gGlCgADIf8AAPjdt6s="
	sDec, _ := base64.StdEncoding.DecodeString(data)
	payload := NewPHYPayload(Proprietary)
	err := payload.UnmarshalBinary(sDec)
	if err != nil {
		t.Fatal(err)
	}
}

// Test decoding of different message types
func TestDecodeMessageTypes(t *testing.T) {
	p := NewPHYPayload(Proprietary)
	// ConfirmedDataUp et al are handled roughly the same
	ordinaryMessage := "YOZy5gGlCgADIf8AAPjdt6s="
	buffer, _ := base64.StdEncoding.DecodeString(ordinaryMessage)
	if err := p.UnmarshalBinary(buffer); err != nil {
		t.Error("Got error unmarshaling binary")
	}
	if p.MACPayload.FHDR.FCnt == 0 {
		t.Error("FCnt should not be set to 0")
	}
	if p.MHDR.MType == JoinRequest {
		t.Error("Did not expect JoinRequest here")
	}
	if p.JoinRequestPayload.DevNonce != 0 {
		t.Error("Did not expect DevNonce to be decoded")
	}

	p = NewPHYPayload(Proprietary)
	// JoinRequest/JoinAccept are handled separately
	joinRequestMessage := "AAgHBgUEAwIBvrrvvr66775+xPtJBQI="
	buffer, _ = base64.StdEncoding.DecodeString(joinRequestMessage)
	if err := p.UnmarshalBinary(buffer); err != nil {
		t.Error("Got error unmarshaling JoinRequest msg: ", err)
	}
	if p.MACPayload.FHDR.FCnt != 0 {
		t.Error("Expected MACPayload to stay unchanged")
	}
	if p.MHDR.MType != JoinRequest {
		t.Error("Not the expected type")
	}

	// Proprietary messages yields errors
	m := MHDR{MajorVersion: 0, MType: Proprietary}
	pos := 0
	if err := m.encode(buffer, &pos); err != nil {
		t.Error("Got error encoding buffer: ", err)
	}

	if err := p.UnmarshalBinary(buffer); err == nil {
		t.Error("Didn't get an error when decoding proprietary message")
	}
}

// -----------------------------------------------------------------------------
// Benchmarks

func BenchmarkDecode_Sample1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data := "YOZy5gGgCQAyiPuX"
		sDec, _ := base64.StdEncoding.DecodeString(data)

		payload := NewPHYPayload(Proprietary)
		payload.UnmarshalBinary(sDec)
	}
}

func BenchmarkDecode_Sample2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data := "gOZy5gGAAAAM29rW3S3Dfag="
		sDec, _ := base64.StdEncoding.DecodeString(data)

		payload := NewPHYPayload(Proprietary)
		payload.UnmarshalBinary(sDec)
	}
}
