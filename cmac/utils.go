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
// Utility functions used by the CMAC generation code

// Shift buffer 1 bit left
func shiftLeft(buf []byte) []byte {
	if len(buf) == 0 {
		return make([]byte, 0)
	}
	ret := make([]byte, len(buf))

	overflow := byte(0)
	for j := len(buf) - 1; j >= 0; j-- {
		ret[j] = (buf[j] << 1)
		ret[j] |= overflow
		overflow = (buf[j] & 0x80) >> 7
	}
	return ret
}

// xor one buffer with another one
func xor(buf1 []byte, buf2 []byte) []byte {
	ret := make([]byte, len(buf1))
	for i := range buf1 {
		ret[i] = buf1[i] ^ buf2[i]
	}
	return ret
}

// pad buffer with a single bit + zero up to the maximum
func padblock(buf []byte, n int) []byte {
	missing := n - len(buf)
	if missing <= 0 {
		return buf
	}
	if missing == 1 {
		return append(buf, 0x80) // 0x80 = 10000000b
	}
	return append(append(buf, 0x80), make([]byte, missing-1)...)
}
