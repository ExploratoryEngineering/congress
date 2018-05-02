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
	"encoding/hex"
	"fmt"
	"strings"
)

// EUI represents an IEEE EUI-64 identifier. The identifier is described at
// http://standards.ieee.org/develop/regauth/tut/eui64.pdf
type EUI struct {
	Octets [8]byte
}

// String returns a string representation of the EUI (XX-XX-XX-XX...)
func (eui EUI) String() string {
	return fmt.Sprintf("%02x-%02x-%02x-%02x-%02x-%02x-%02x-%02x",
		eui.Octets[0], eui.Octets[1], eui.Octets[2], eui.Octets[3],
		eui.Octets[4], eui.Octets[5], eui.Octets[6], eui.Octets[7])
}

// EUIFromString converts a string on the format "xx-xx-xx..." in hex to an
// internal representation
func EUIFromString(euiStr string) (EUI, error) {
	tmpBuf, err := hex.DecodeString(strings.TrimSpace(strings.Replace(euiStr, "-", "", -1)))
	if err != nil {
		return EUI{}, err
	}
	if len(tmpBuf) != 8 {
		return EUI{}, ErrInvalidParameterFormat
	}
	ret := EUI{}
	copy(ret.Octets[:], tmpBuf)
	return ret, nil
}

// EUIFromUint64 converts an uint64 value to an EUI.
func EUIFromUint64(val uint64) EUI {
	ret := EUI{}
	for i := 7; i >= 0; i-- {
		ret.Octets[i] = byte(val & 0xFF)
		val >>= 8
	}
	return ret
}

// ToUint64 returns the EUI as a uin64 integer
func (eui *EUI) ToUint64() uint64 {
	ret := uint64(0)
	for i := 0; i < 8; i++ {
		ret <<= 8
		ret += uint64(eui.Octets[i])
	}
	return ret
}
