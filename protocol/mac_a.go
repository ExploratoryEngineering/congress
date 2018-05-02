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
)

// MACLinkCheckReq is sent from the end-device to the network server
type MACLinkCheckReq struct {
	macBase
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACLinkCheckReq) Length() int {
	return 1
}

func (m *MACLinkCheckReq) encode(buffer []byte, pos *int) error {
	return encodeID(m, buffer, pos)
}

func (m *MACLinkCheckReq) decode(buffer []byte, pos *int) error {
	return decodeID(m, buffer, pos)
}

// MACLinkCheckAns is sent from the network server to the end-device
type MACLinkCheckAns struct {
	macBase
	Margin uint8
	GwCnt  uint8
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACLinkCheckAns) Length() int {
	return 3
}

func (m *MACLinkCheckAns) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	buffer[*pos] = byte(m.Margin)
	*pos++
	buffer[*pos] = byte(m.GwCnt)
	*pos++
	return nil
}

func (m *MACLinkCheckAns) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.Margin = buffer[*pos]
	*pos++
	m.GwCnt = buffer[*pos]
	*pos++
	return nil
}

// MACLinkADRReq is sent from the network server to the end-device to perform data rate adoption
type MACLinkADRReq struct {
	macBase
	DataRate   uint8  // Region specific - see [7.1.3], [7.2.3], [7.3.3], [7.4.3]
	TXPower    uint8  // Region specific - see [7.1.3], [7.2.3], [7.3.3], [7.4.3]
	ChMask     uint16 // Channel mask - 1 bit per channel
	Redundancy uint8  // TODO: See [5.2]
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACLinkADRReq) Length() int {
	return 5
}

func (m *MACLinkADRReq) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	buffer[*pos] = byte((m.DataRate << 4) | (m.TXPower & 0x0F))
	*pos++
	binary.LittleEndian.PutUint16(buffer[*pos:], m.ChMask)
	*pos += 2
	buffer[*pos] = byte(m.Redundancy)
	*pos++
	return nil
}

func (m *MACLinkADRReq) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.DataRate = (buffer[*pos] & 0xF0) >> 4
	m.TXPower = buffer[*pos] & 0x0F
	*pos++
	m.ChMask = binary.LittleEndian.Uint16(buffer[*pos:])
	*pos += 2
	m.Redundancy = buffer[*pos]
	*pos++
	return nil
}

// MACLinkADRAns is sent from the end-device to the network server as a response to the LinkAdrReq command
type MACLinkADRAns struct {
	macBase
	PowerACK       bool
	DataRateACK    bool
	ChannelMaskACK bool
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACLinkADRAns) Length() int {
	return 2
}

func (m *MACLinkADRAns) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	status := 0
	if m.PowerACK {
		status |= (1 << 2)
	}
	if m.DataRateACK {
		status |= (1 << 1)
	}
	if m.ChannelMaskACK {
		status |= (1 << 0)
	}
	buffer[*pos] = byte(status)
	*pos++
	return nil
}

func (m *MACLinkADRAns) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.PowerACK = buffer[*pos]&0x4 != 0
	m.DataRateACK = buffer[*pos]&0x2 != 0
	m.ChannelMaskACK = buffer[*pos]&0x1 != 0
	*pos++
	return nil
}

// MACDutyCycleReq is sent from the network coordinator to the end-device to set the duty cycle
type MACDutyCycleReq struct {
	macBase
	MaxDCycle uint8
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACDutyCycleReq) Length() int {
	return 2
}

func (m *MACDutyCycleReq) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	buffer[*pos] = byte(m.MaxDCycle)
	*pos++
	return nil
}

func (m *MACDutyCycleReq) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.MaxDCycle = buffer[*pos]
	*pos++
	return nil
}

// MACDutyCycleAns is sent from the end-device as a response to the DutyCycleReq command
type MACDutyCycleAns struct {
	macBase
	// No payload
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACDutyCycleAns) Length() int {
	return 1
}

func (m *MACDutyCycleAns) encode(buffer []byte, pos *int) error {
	return encodeID(m, buffer, pos)
}

func (m *MACDutyCycleAns) decode(buffer []byte, pos *int) error {
	return decodeID(m, buffer, pos)
}

// MACRXParamSetupReq is sent from the network to the end-device
type MACRXParamSetupReq struct {
	macBase
	RX1DRoffset uint8
	RX2DataRate uint8
	Frequency   uint32
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACRXParamSetupReq) Length() int {
	return 5
}

func (m *MACRXParamSetupReq) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	dlsettings := byte(0)
	dlsettings |= (m.RX1DRoffset & 0x7) << 4 // 3 bits for RX1DRoffset
	dlsettings |= (m.RX2DataRate & 0xF)      // ..and 4 bits for RX2DataRate

	buffer[*pos] = dlsettings
	*pos++

	buffer[*pos] = byte(m.Frequency & 0x0000FF)
	*pos++
	buffer[*pos] = byte((m.Frequency & 0x00FF00) >> 8)
	*pos++
	buffer[*pos] = byte((m.Frequency & 0xFF0000) >> 16)
	*pos++
	return nil
}

func (m *MACRXParamSetupReq) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.RX1DRoffset = uint8(buffer[*pos]&0x70) >> 4
	m.RX2DataRate = uint8(buffer[*pos] & 0x0F)
	*pos++
	m.Frequency = uint32(buffer[*pos]) + uint32(buffer[*pos+1])<<8 + uint32(buffer[*pos+2])<<16
	*pos += 3
	return nil
}

// MACRXParamSetupAns is sent from the end-device to the network in response to the RXParamSetupReq
type MACRXParamSetupAns struct {
	macBase
	RX1DRoffsetACK bool
	RX2DataRateACK bool
	ChannelACK     bool
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACRXParamSetupAns) Length() int {
	return 2
}

func (m *MACRXParamSetupAns) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	status := byte(0)
	if m.RX1DRoffsetACK {
		status |= (1 << 2)
	}
	if m.RX2DataRateACK {
		status |= (1 << 1)
	}
	if m.ChannelACK {
		status |= 1
	}
	buffer[*pos] = status
	*pos++
	return nil
}

func (m *MACRXParamSetupAns) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.RX1DRoffsetACK = buffer[*pos]&0x04 != 0
	m.RX2DataRateACK = buffer[*pos]&0x02 != 0
	m.ChannelACK = buffer[*pos]&0x01 != 0
	*pos++
	return nil
}

// MACDevStatusReq is sent from the network server to an end-device to request the device status
type MACDevStatusReq struct {
	macBase
	// No payload
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACDevStatusReq) Length() int {
	return 1
}

func (m *MACDevStatusReq) encode(buffer []byte, pos *int) error {
	return encodeID(m, buffer, pos)
}

func (m *MACDevStatusReq) decode(buffer []byte, pos *int) error {
	return decodeID(m, buffer, pos)
}

// Battery power status constants
const (
	BatteryExternalPower uint8 = 0
	BatteryUnavailable   uint8 = 0xFF
)

// MACDevStatusAns is sent from the end-device to the network server as a response to the DevStatusReq command
type MACDevStatusAns struct {
	macBase
	Battery uint8
	Margin  uint8
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACDevStatusAns) Length() int {
	return 3
}

func (m *MACDevStatusAns) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	buffer[*pos] = byte(m.Battery)
	*pos++
	buffer[*pos] = byte(m.Margin & 0x3F) // aka b000111111
	*pos++
	return nil
}

func (m *MACDevStatusAns) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.Battery = buffer[*pos]
	*pos++
	m.Margin = buffer[*pos]
	*pos++
	return nil
}

// MACNewChannelReq is sent from the network server to set up a new channel on the device
type MACNewChannelReq struct {
	macBase
	ChIndex uint8
	Freq    uint32
	MaxDR   uint8
	MinDR   uint8
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACNewChannelReq) Length() int {
	return 6
}
func (m *MACNewChannelReq) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	buffer[*pos] = byte(m.ChIndex)
	*pos++

	binary.LittleEndian.PutUint32(buffer[*pos:], (m.Freq & 0x00FFFFFF))
	*pos += 3

	drRange := (m.MaxDR&0x0F)<<4 | (m.MinDR & 0x0F)
	buffer[*pos] = drRange
	*pos++

	return nil
}

func (m *MACNewChannelReq) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.ChIndex = buffer[*pos]
	*pos++
	m.Freq = binary.LittleEndian.Uint32(buffer[*pos:]) & 0x00FFFFFF
	*pos += 3
	m.MaxDR = (buffer[*pos] & 0xF0) >> 4
	m.MinDR = buffer[*pos] & 0x0F
	*pos++
	return nil
}

// MACNewChannelAns is sent from the end-device to the network server as a response to the NewChannelReq command
type MACNewChannelAns struct {
	macBase
	DataRangeOK        bool
	ChannelFrequencyOK bool
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACNewChannelAns) Length() int {
	return 2
}

func (m *MACNewChannelAns) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	val := byte(0)
	if m.DataRangeOK {
		val |= (1 << 1)
	}
	if m.ChannelFrequencyOK {
		val |= (1 << 0)
	}
	buffer[*pos] = val
	*pos++
	return nil
}

func (m *MACNewChannelAns) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.DataRangeOK = buffer[*pos]&0x02 != 0
	m.ChannelFrequencyOK = buffer[*pos]&0x01 != 0
	*pos++
	return nil
}

// MACRXTimingSetupReq is sent by the network server to an end-device to set up delay between RX and TX.
type MACRXTimingSetupReq struct {
	macBase
	Del uint8 // [5.7]
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACRXTimingSetupReq) Length() int {
	return 2
}

func (m *MACRXTimingSetupReq) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	buffer[*pos] = byte(m.Del & 0x0F)
	*pos++

	return nil
}

func (m *MACRXTimingSetupReq) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.Del = buffer[*pos] & 0xF
	*pos++
	return nil
}

// MACRXTimingSetupAns is sent by the device to acknowledge a RXTimingSetupReq message
type MACRXTimingSetupAns struct {
	macBase
	// no payload
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACRXTimingSetupAns) Length() int {
	return 1
}

func (m *MACRXTimingSetupAns) encode(buffer []byte, pos *int) error {
	return encodeID(m, buffer, pos)
}

func (m *MACRXTimingSetupAns) decode(buffer []byte, pos *int) error {
	return decodeID(m, buffer, pos)
}
