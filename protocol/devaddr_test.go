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
	"encoding/binary"
	"testing"
)

func TestDevAddrUintConversion(t *testing.T) {
	addr := DevAddr{NwkID: 0x7F, NwkAddr: 0x0F0E0D}
	val := addr.ToUint32()

	otherAddr := DevAddrFromUint32(val)
	if addr != otherAddr {
		t.Fatalf("Values do not match: %v != %v", addr, otherAddr)
	}
}

func TestDevAddrStringConversion(t *testing.T) {
	addr := DevAddr{NwkID: 0x1, NwkAddr: 0x0A0B0C}

	str := addr.String()
	newAddr, err := DevAddrFromString(str)
	if err != nil {
		t.Fatal("Got error converting: ", err)
	}
	if addr != newAddr {
		t.Fatalf("DevAddr not equal: %x != %x", addr.NwkAddr, newAddr.NwkAddr)
	}

	_, err = DevAddrFromString("go go gorillaz")
	if err == nil {
		t.Fatal("Expected error when converting invalid string (not hex) but didn't get one")
	}

	_, err = DevAddrFromString("010203040506070809")
	if err == nil {
		t.Fatal("Expected error when converting invalid string (> 32 bit) but didn't get one")
	}
}

func TestDevAddrEncoding(t *testing.T) {
	addr := DevAddr{NwkID: 0x2, NwkAddr: 0x01234}
	buffer := make([]byte, 4)

	pos := 0
	if err := addr.encode(buffer, &pos); err != nil {
		t.Fatalf("Got error encoding DevAddr: %v", err)
	}
	if pos != 4 {
		t.Fatalf("Position counter not updated")
	}
	// Read little endian uint32 and compare with original
	val := binary.LittleEndian.Uint32(buffer[:])
	addr2 := DevAddrFromUint32(val)
	if addr != addr2 {
		t.Fatalf("Original and decoded does not match: %v != %v", addr, addr2)
	}

	// Test with a buffer that is too small to hold everything
	var tinyBuf [1]byte
	if err := addr.encode(tinyBuf[:], &pos); err == nil {
		t.Fatal("Expected error when encoding into 1-byte buffer but got none")
	}
}

func TestDevAddrEncodeInvalidInput(t *testing.T) {
	d1 := DevAddr{NwkID: 0xFF, NwkAddr: 0}
	d2 := DevAddr{NwkID: 0, NwkAddr: 0xFFFFFFFF}
	buffer := make([]byte, 10)
	pos := 0
	if err := d1.encode(buffer, &pos); err == nil {
		t.Fatal("Expected error with invalid NwkID")
	}
	if err := d2.encode(buffer, &pos); err == nil {
		t.Fatal("Expected error with invalid NwkAddr")
	}
}
func TestDevAddrEncodeDecode(t *testing.T) {
	d1 := NewDevAddr()
	buffer := make([]byte, 10)
	pos := 0
	if err := d1.encode(buffer, &pos); err != nil {
		t.Fatalf("Got error encoding DevAddr: %v", err)
	}
	pos = 0
	d2 := DevAddr{}
	if err := d2.decode(buffer, &pos); err != nil {
		t.Fatalf("Got error decoding DevAddr: %v", err)
	}
	if d1 != d2 {
		t.Fatalf("Encoded and decoded are different: %+v != %+v", d1, d2)
	}
}

func TestDevAddrBufferRangeChecks(t *testing.T) {
	basicEncoderTests(t, &DevAddr{})
	basicDecoderTests(t, &DevAddr{})
}
