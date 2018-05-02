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
// MType is the message type
type MType uint8

const (
	// JoinRequest is sent by the end-device when it wants to do OTAA [4.2.1]
	JoinRequest MType = 0
	// JoinAccept is sent by the network when it accepts a JoinRequest from an end-device [4.2.1]
	JoinAccept MType = 1
	// UnconfirmedDataUp is sent by the end device [4.2.1]
	UnconfirmedDataUp MType = 2
	// UnconfirmedDataDown is sent by the network [4.2.1]
	UnconfirmedDataDown MType = 3
	// ConfirmedDataUp is sent by the end-device [4.2.1]
	ConfirmedDataUp MType = 4
	// ConfirmedDataDown is sent by the network [4.2.1]
	ConfirmedDataDown MType = 5
	// RFU is - surprisingly - Reserved for Future Use [4.2.1]
	RFU MType = 6
	// Proprietary is a message type used when implementing proprietary messages [4.2.1]
	Proprietary MType = 7
)

func (m MType) String() string {
	switch m {
	case JoinRequest:
		return "JoinRequest"
	case JoinAccept:
		return "JoinAccept"
	case UnconfirmedDataUp:
		return "UnconfirmedDataUp"
	case UnconfirmedDataDown:
		return "UnconfirmedDataDown"
	case ConfirmedDataUp:
		return "ConfirmedDataUp"
	case ConfirmedDataDown:
		return "ConfirmedDataDown"
	case RFU:
		return "RFU"
	case Proprietary:
		return "Proprietary"
	}
	return "[Unknown type]"
}

// Uplink returns true if the message type is an uplink. RFU and Proprietary messages
// are considered uplink messages
func (t MType) Uplink() bool {
	return (t == JoinRequest || t == UnconfirmedDataUp || t == ConfirmedDataUp)
}

const (
	// LoRaWANR1 is the official version number used in the MHDR struct [4.2.2]
	LoRaWANR1 uint8 = 0
	// MaxSupportedVersion is the latest version supported
	MaxSupportedVersion uint8 = LoRaWANR1
)

// MHDR is the message header [4.2]
type MHDR struct {
	MType        MType // [4.2.1]
	MajorVersion uint8 // [4.2.2]
}

// decode extracts the MHDR struct from a byte array
func (m *MHDR) decode(buffer []byte, pos *int) error {
	const MessageTypeMask byte = 0xE0 // bit 7..5
	const MajorVersionMask byte = 0x3

	val := buffer[*pos]
	*pos++
	m.MType = MType((val & MessageTypeMask) >> 5)

	m.MajorVersion = val & MajorVersionMask
	if m.MajorVersion > MaxSupportedVersion {
		return ErrInvalidLoRaWANVersion
	}
	return nil
}

func (m *MHDR) encode(buffer []byte, count *int) error {
	// Bits 5-7 are message type, bits 0-1 are version. Rest is 0
	buffer[*count] = ((byte(m.MType) & 0x7) << 5) | (byte(m.MajorVersion) & 0x3)
	*count++
	return nil
}
