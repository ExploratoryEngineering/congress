package cmac

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
	"bytes"
	"testing"
)

func testWithNBytes(n int, t *testing.T) {
	buf := bytes.Repeat([]byte{0x55}, n)
	newbuf := shiftLeft(buf)
	if len(newbuf) != len(buf) {
		t.Errorf("Expected array to be %d bytes, got %d", len(buf), len(newbuf))
	}

	for index, val := range newbuf {
		if val != 0xAA {
			t.Errorf("Expected byte at %d to be 0x%02x but it was 0x%02x", index, 0xAA, val)
		}
	}
}

// Test shifting with 0..1000 bytes
func TestShiftLeftMultipleBytes(t *testing.T) {
	for n := 0; n < 1000; n++ {
		testWithNBytes(n, t)
	}
}

func TestShiftLeftBytePattern(t *testing.T) {
	buf := bytes.Repeat([]byte{0xFF, 0x00}, 8)
	// byte pattern is 1111111100000001111111000000
	// Shifting left 8 times will invert the pattern
	for i := 0; i < 8; i++ {
		buf = shiftLeft(buf)
	}
	for j := 0; j < 16; j += 2 {
		if buf[j] != 0x00 && buf[j+1] != 0xFF {
			t.Errorf("Mismatch at position %d", j)
		}
	}
}
func testXor(n int, t *testing.T) {
	buf1 := bytes.Repeat([]byte{0x55}, n)
	buf2 := bytes.Repeat([]byte{0xFF}, n)

	ret := xor(buf1, buf2)

	if len(ret) != n {
		t.Errorf("Expected %d bytes back, got %d", n, len(ret))
	}
	for i := range ret {
		if ret[i] != 0xAA {
			t.Errorf("Expected 0x%02x at index %d but got 0x%02x", 0xAA, i, ret[i])
		}
	}
}

// Test buffers from 0 to 1000 bytes long
func TestXor(t *testing.T) {
	for n := 0; n < 1000; n++ {
		testXor(n, t)
	}
}

func testNPadding(n int, t *testing.T) {
	buf := bytes.Repeat([]byte{0xFF}, 10)

	padded := padblock(buf, n)

	if len(padded) != n {
		t.Errorf("Padding didn't work. Expected %d got %d", n, len(padded))
	}
}

func TestPadding(t *testing.T) {
	var constZero = []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}

	for i := 10; i < len(constZero); i++ {
		testNPadding(i, t)
	}
}
