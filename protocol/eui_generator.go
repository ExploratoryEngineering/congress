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
import "fmt"

// MASize is the defined sizes. There are three different sizes:
// MA-L (large), MA-M (medium) and MA-S (small)
type MASize uint8

// EUI size specifiers
const (
	MALarge  MASize = 24 // Append 40 bits to get EUI-64
	MAMedium MASize = 28 // Append 36 bits to get EUI-64
	MASmall  MASize = 36 // Append 28 bits to get EUI-64
)

// MA defines a MA block
type MA struct {
	Prefix [5]byte // MA prefix for EUI
	Size   MASize  // The size of the prefix bytes
}

// Combine combines the MA with an EUI to make the final EUI.
func (m MA) Combine(eui EUI) EUI {
	ret := EUI{}
	for i := 0; i < 3; i++ {
		ret.Octets[i] = m.Prefix[i]
	}
	var startingIndex int
	switch m.Size {
	case MALarge:
		startingIndex = 3
	case MAMedium:
		ret.Octets[3] = (m.Prefix[3] & 0xF0) | (eui.Octets[3] & 0x0F)
		startingIndex = 4
	case MASmall:
		ret.Octets[3] = m.Prefix[3]
		ret.Octets[4] = (m.Prefix[4] & 0xF0) | (eui.Octets[4] & 0x0F)
		startingIndex = 5
	}
	for i := startingIndex; i < 8; i++ {
		ret.Octets[i] = eui.Octets[i]
	}
	return ret
}

// String prints the MA as a hex-formatted string. Depending on the size the string will be
// "hh-hh-hh" (MA-L), "hh-hh-hh-hh" (MA-M) or "hh-hh-hh-hh-hh" (MA-S)
func (m *MA) String() string {
	switch m.Size {
	case MALarge:
		return fmt.Sprintf("%02x-%02x-%02x", m.Prefix[0], m.Prefix[1], m.Prefix[2])
	case MAMedium:
		return fmt.Sprintf("%02x-%02x-%02x-%02x", m.Prefix[0], m.Prefix[1], m.Prefix[2], m.Prefix[3])
	default:
		return fmt.Sprintf("%02x-%02x-%02x-%02x-%02x", m.Prefix[0], m.Prefix[1], m.Prefix[2], m.Prefix[3], m.Prefix[4])

	}
}

// NewMA creates a new MA block. The size of the prefix decides which kind of
// MA this is; 3 bytes yields a MA-L (24 bits, all three bytes are used),
// 4 bytes MA-M (28 bits, only the 4 MSB are used of the last byte) and 5 bytes
// yields a MA-S (36 bits, only the 4 MSB are used of the 5th byte)
func NewMA(prefix []byte) (MA, error) {
	ret := MA{}
	switch len(prefix) {
	case 3:
		ret.Size = MALarge
	case 4:
		ret.Size = MAMedium
	case 5:
		ret.Size = MASmall
	default:
		return MA{}, ErrInvalidParameterFormat
	}
	for i := range prefix {
		ret.Prefix[i] = prefix[i]
	}
	return ret, nil
}

// Telenor owns some MAC assignments
//    * MA-L: 00-09-09, Telenor Connect assigned
//    * MA-S 70-B3-D5-(27D000 - 27DFFF). Telenor Connexion AB)

// NewDeviceEUI creates a new device EUI. The following bit pattern will be used
// for device EUIs:
//     8.......7.......6.......5.......4.......3.......2.......1.......0
//     |                               |NwkID--|                         7 bits
//     |                                       |NwkAddr (counter)------| 25 bits
//     |              |NetID-------------------|                         24 bits
//     |MA-L-------------------|                                         24 bits
//     |MA-M-----------------------|                                     28 bits
//     |MA-S------------------------------|                              36 bits
//
// The MA block might overwrite parts of the NetID and NwkID values.
func NewDeviceEUI(ma MA, netID uint32, nwkAddr uint32) EUI {
	return ma.Combine(EUIFromUint64(uint64(netID)<<25 | uint64(nwkAddr)))
}

// NewApplicationEUI creates a new application EUI. The bit layout for the EUI
// is as follows:
//     8.......7.......6.......5.......4.......3.......2.......1.......0
//     |                               |NwkID--|                         7 bits
//     |                                       |(Counter)--------------| 25 bits
//     |              |NetID-------------------|                         24 bits
//     |MA-L-------------------|                                         24 bits
//     |MA-M-----------------------|                                     28 bits
//     |MA-S------------------------------|                              36 bits
//
// Both NetID and NwkID might be overwritten by the MA block.
func NewApplicationEUI(ma MA, netID uint32, counter uint32) EUI {
	return ma.Combine(EUIFromUint64(uint64(netID)<<25 | uint64(counter)))
}

// NewNetworkEUI creates a new network EUI. The bit layout is as follows:
//     8.......7.......6.......5.......4.......3.......2.......1.......0
//     |                               |NwkID--|                         7 bits
//     |                                       |(0)--------------------| 25 bits
//     |              |NetID (counter)---------|                         24 bits
//     |MA-L-------------------|                                         24 bits
//     |MA-M-----------------------|                                     28 bits
//     |MA-S------------------------------|                              36 bits
//
// The medium and small MA blocks might overwrite the NetID and NwkID values.
func NewNetworkEUI(ma MA, netID uint32) EUI {
	return ma.Combine(EUIFromUint64(uint64(netID) << 25))
}

// This is the maximum number of bits that can be used for a NetID in a network EUI
const (
	MaxNetworkBitsMAL uint32 = 0x07FFF // 64 - MALarge (24) - 25 = 15 bits available
	MaxNetworkBitsMAM uint32 = 0x07FF  // 64 - MAMedium (28) - 25 = 11 bits available
	MaxNetworkBitsMAS uint32 = 0x07    // 64 - MASmall (36) - 25 = 3 bits available
)

// MaxNetID return the maximum NetID that can safely be put into the EUI
func MaxNetID(size MASize) uint32 {
	switch size {
	case MASmall:
		return MaxNetworkBitsMAS
	case MAMedium:
		return MaxNetworkBitsMAM
	default:
		return MaxNetworkBitsMAL
	}
}
