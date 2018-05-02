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
	"fmt"
	"math/rand"
	"strconv"
)

const (
	// MaxNwkID is the maximum allowed value for NwkID
	MaxNwkID = 0x7F
	// MaxNwkAddr is the maximum allowed value for NwkAddr
	MaxNwkAddr = 0x1FFFFFF
	// NetworkIDMask is the
	NetworkIDMask uint32 = 0xFE000000
)

// DevAddr is a device address [6.1.1]
type DevAddr struct {
	NwkID   uint8  // 7 bits [6.1.1]
	NwkAddr uint32 // 25 bits [6.1.1]

}

// ToUint32 converts the device address as a single 32-bit integer
func (d DevAddr) ToUint32() uint32 {
	return (uint32(d.NwkID)<<25 | (uint32(d.NwkAddr) & MaxNwkAddr))
}

// DevAddrFromUint32 converts the integer into a DevAddr struct
func DevAddrFromUint32(val uint32) DevAddr {
	return DevAddr{
		NwkID:   uint8((val >> 25) & MaxNwkID),
		NwkAddr: uint32(val & MaxNwkAddr),
	}
}

// String prints a string representation of the device address (aka the fmt.Stringer interface)
func (d DevAddr) String() string {
	return fmt.Sprintf("%08x", d.ToUint32())
}

// DevAddrFromString converts a hex representation to a DevAddr value
func DevAddrFromString(devAddrStr string) (DevAddr, error) {
	val, err := strconv.ParseInt(devAddrStr, 16, 32)
	if err != nil {
		return DevAddr{}, err
	}
	return DevAddrFromUint32(uint32(val)), nil
}

const defaultNwkID = uint8(0)

// NewDevAddr creates a new random DevAddr value. It uses the defalt
// pseudo-random generator.
func NewDevAddr() DevAddr {
	return DevAddr{defaultNwkID, rand.Uint32() & 0x1FFFFFF}
}

// Encode the DevAddr type into a buffer. A minimum of 4 bytes is required.
func (d *DevAddr) encode(buffer []byte, pos *int) error {
	if pos == nil {
		return ErrNilError
	}
	if len(buffer) < (*pos + 4) {
		return ErrBufferTruncated
	}
	if d.NwkID > MaxNwkID || d.NwkAddr > MaxNwkAddr {
		return ErrParameterOutOfRange
	}
	binary.LittleEndian.PutUint32(buffer[*pos:], d.ToUint32())
	*pos += 4
	return nil
}

// decodeDeviceAddress extracts NetworkId and NetworkAddress
func (d *DevAddr) decode(octets []byte, pos *int) error {
	if pos == nil {
		return ErrNilError
	}
	if len(octets) < (*pos + 4) {
		return ErrBufferTruncated
	}
	var fullAddress = binary.LittleEndian.Uint32(octets[*pos:])
	d.NwkID = uint8((fullAddress & NetworkIDMask) >> 25)
	d.NwkAddr = fullAddress & MaxNwkAddr

	*pos += 4
	return nil
}
