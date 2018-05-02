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
// General MAC Command declarations

// CID represents a MACCommand [5]
type CID uint8

// MAC commands for Class A devices
const (
	// LinkCheckReq is sent by the end-device to the network (no payload).
	LinkCheckReq CID = 0x02
	// LinkCheckAns is sent by the network to the end-device.
	LinkCheckAns CID = 0x02
	// LinkADRReq is sent by the network to the end-device.
	LinkADRReq CID = 0x03
	// LinkADRAns is sent by the end-device to the network.
	LinkADRAns CID = 0x03
	// DutyCycleReq is sent by the network to the end-device.
	DutyCycleReq CID = 0x04
	// DutyCycleAns is sent by the end-device to the network.
	DutyCycleAns CID = 0x04
	// RXParamSetupReq is sent by the network to the end-device (no payload).
	RXParamSetupReq CID = 0x05
	// RXParamSetupAns is sent by the end-device to the network.
	RXParamSetupAns CID = 0x05
	// DevStatusReq is sent by the network to the end-device (no payload).
	DevStatusReq CID = 0x06
	// DevStatusAns is sent by the end-device to the network.
	DevStatusAns CID = 0x06
	// NewChannelReq is sent by the network to the end-device.
	NewChannelReq CID = 0x07
	// NewChannelAns is sent by the end-device to the network.
	NewChannelAns CID = 0x07
	// RXTimingSetupReq is sent by the network to the end-device.
	RXTimingSetupReq CID = 0x08
	// RXTimingSetupAns is sent by the end-device to the network (no payload)
	RXTimingSetupAns CID = 0x08
)

// MAC commands for Class B devices
const (
	// PingSlotInfoReq is sent by the end-device to the network.
	PingSlotInfoReq CID = 0x10
	// PingSlotInfoAns is sent by the network to the end-device.
	PingSlotInfoAns CID = 0x10
	// PingSlotChannelReq is sent by the network to the end-device.
	PingSlotChannelReq CID = 0x11
	// PingSlotFreqAns is sent by the end-device to the network.
	PingSlotFreqAns CID = 0x11
	// BeaconTimingReq is sent by the end-device to the network.
	BeaconTimingReq CID = 0x12
	// BeaconTimingAns is sent by the network to the end-device.
	BeaconTimingAns CID = 0x12
	// BeaconFreqReq is sent by the network to the end-device.
	BeaconFreqReq CID = 0x13
	// BeaconFreqAns is sent by the end-device to the network.
	BeaconFreqAns CID = 0x13
)

// MACCommand represents MAC commands
type MACCommand interface {
	// ID returns the Command ID (aka CID) for MAC command.
	ID() CID
	// Length returns the length of the MAC command when encoded
	Length() int
	// Uplink returns true if this command is an uplink command
	Uplink() bool
	// Encode into buffer at specified position. Includes identifier
	encode(buffer []byte, pos *int) error
	// Decode MAC command at position. Includes identifier
	decode(buffer []byte, pos *int) error
}

// Implemented by all of the mac commands.
type macBase struct {
	id     CID
	uplink bool
}

func (m *macBase) ID() CID {
	return m.id
}

func (m *macBase) Uplink() bool {
	return m.uplink
}

// Check needed buffer size vs actual size. Returns true if the position
// is inside the buffer and the buffer is big enough to hold all of the required data.
func isValidBuffer(buffer []byte, pos *int, cmd MACCommand) bool {
	if len(buffer) < *pos+cmd.Length() {
		return false
	}
	return true
}

// This is the default implementation for empty payloads; only the CID is encoded.
func encodeID(cmd MACCommand, buffer []byte, pos *int) error {
	if pos == nil {
		return ErrNilError
	}
	if !isValidBuffer(buffer, pos, cmd) {
		return ErrBufferTruncated
	}
	buffer[*pos] = byte(cmd.ID())
	*pos++
	return nil
}

// This is the default implementation for empty payloads. Only the CID is decoded.
func decodeID(cmd MACCommand, buffer []byte, pos *int) error {
	if pos == nil {
		return ErrNilError
	}
	if !isValidBuffer(buffer, pos, cmd) {
		return ErrBufferTruncated
	}
	if buffer[*pos] != byte(cmd.ID()) {
		return ErrInvalidSource
	}
	*pos++
	return nil
}

// NewUplinkMACCommand creates a new MAC Command instance
func NewUplinkMACCommand(id CID) MACCommand {
	switch id {
	case LinkCheckReq:
		return &MACLinkCheckReq{macBase{LinkCheckReq, true}}
	case LinkADRAns:
		return &MACLinkADRAns{macBase{LinkADRAns, true}, false, false, false}
	case DutyCycleAns:
		return &MACDutyCycleAns{macBase{DutyCycleAns, true}}
	case RXParamSetupAns:
		return &MACRXParamSetupAns{macBase{RXParamSetupAns, true}, false, false, false}
	case DevStatusAns:
		return &MACDevStatusAns{macBase{DevStatusAns, true}, 0, 0}
	case NewChannelAns:
		return &MACNewChannelAns{macBase{NewChannelAns, true}, false, false}
	case RXTimingSetupAns:
		return &MACRXTimingSetupAns{macBase{RXTimingSetupAns, true}}
	case PingSlotInfoReq:
		return &MACPingSlotInfoReq{macBase{PingSlotInfoReq, true}, 0, 0}
	case PingSlotFreqAns:
		return &MACPingSlotFreqAns{macBase{PingSlotFreqAns, true}, false, false}
	case BeaconTimingReq:
		return &MACBeaconTimingReq{macBase{BeaconTimingReq, true}}
	case BeaconFreqAns:
		return &MACBeaconFreqAns{macBase{BeaconFreqAns, true}}
	}
	return nil
}

// NewDownlinkMACCommand returns a new downlink command
func NewDownlinkMACCommand(id CID) MACCommand {
	switch id {
	case LinkCheckAns:
		return &MACLinkCheckAns{macBase{LinkCheckAns, false}, 0, 0}
	case LinkADRReq:
		return &MACLinkADRReq{macBase{LinkADRReq, false}, 0, 0, 0, 0}
	case DutyCycleReq:
		return &MACDutyCycleReq{macBase{DutyCycleReq, false}, 0}
	case RXParamSetupReq:
		return &MACRXParamSetupReq{macBase{RXParamSetupReq, false}, 0, 0, 0}
	case DevStatusReq:
		return &MACDevStatusReq{macBase{DevStatusReq, false}}
	case NewChannelReq:
		return &MACNewChannelReq{macBase{NewChannelReq, false}, 0, 0, 0, 0}
	case RXTimingSetupReq:
		return &MACRXTimingSetupReq{macBase{RXTimingSetupReq, false}, 0}
	case PingSlotInfoAns:
		return &MACPingSlotInfoAns{macBase{PingSlotInfoAns, false}}
	case PingSlotChannelReq:
		return &MACPingSlotChannelReq{macBase{PingSlotChannelReq, false}, 0, 0, 0}
	case BeaconTimingAns:
		return &MACBeaconTimingAns{macBase{BeaconTimingAns, false}, 0, 0}
	case BeaconFreqReq:
		return &MACBeaconFreqReq{macBase{BeaconFreqReq, false}, 0}
	}
	return nil
}
