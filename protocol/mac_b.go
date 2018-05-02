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
import "encoding/binary"

// MACPingSlotInfoReq is sent by the end-device to communicate its data rate and periodicity to the network server [14.1]
type MACPingSlotInfoReq struct {
	macBase
	Periodicity uint8
	DataRate    uint8
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACPingSlotInfoReq) Length() int {
	return 2
}

func (m *MACPingSlotInfoReq) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	buffer[*pos] = byte((m.Periodicity&0x07)<<4) | (m.DataRate & 0x0F)
	*pos++
	return nil
}

func (m *MACPingSlotInfoReq) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.Periodicity = (buffer[*pos] & 0x70) >> 4
	m.DataRate = buffer[*pos] & 0x0F
	*pos++
	return nil
}

// MACPingSlotInfoAns is sent by the network server to the end-device to acknowledge a PingSlotInfoReq command
type MACPingSlotInfoAns struct {
	macBase
	// No payload
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACPingSlotInfoAns) Length() int {
	return 1
}

func (m *MACPingSlotInfoAns) encode(buffer []byte, pos *int) error {
	return encodeID(m, buffer, pos)
}

func (m *MACPingSlotInfoAns) decode(buffer []byte, pos *int) error {
	return decodeID(m, buffer, pos)
}

// MACBeaconFreqReq is sent by the network server to the end-device to modify the frequency
type MACBeaconFreqReq struct {
	macBase
	Frequency uint32 // 3 byte long
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACBeaconFreqReq) Length() int {
	return 4
}

func (m *MACBeaconFreqReq) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, m.Frequency)

	copy(buffer[*pos:*pos+3], buf[0:3])
	*pos += 3
	return nil
}

func (m *MACBeaconFreqReq) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.Frequency = uint32(buffer[*pos+2])<<16 + uint32(buffer[*pos+1])<<8 + uint32(buffer[*pos])
	*pos += 3
	return nil
}

// MACBeaconFreqAns is sent by the end-device to acknowledge a BeaconFreqReq command
type MACBeaconFreqAns struct {
	macBase
	// Payload is assumed to be empty. The spec doesn't mention it but does
	// mention "return an error otherwise". Nobody knows where that error goes. [14.2]
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACBeaconFreqAns) Length() int {
	return 1
}

func (m *MACBeaconFreqAns) encode(buffer []byte, pos *int) error {
	return encodeID(m, buffer, pos)
}

func (m *MACBeaconFreqAns) decode(buffer []byte, pos *int) error {
	return decodeID(m, buffer, pos)
}

// MACPingSlotChannelReq is sent by the server to the end-device to modify the
// frequency down-link pings are sent at
type MACPingSlotChannelReq struct {
	macBase
	Frequency uint32
	MaxDR     uint8
	MinDR     uint8
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACPingSlotChannelReq) Length() int {
	return 5
}

func (m *MACPingSlotChannelReq) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	buffer[*pos] = byte(m.Frequency & 0x0000FF)
	*pos++
	buffer[*pos] = byte((m.Frequency & 0x00FF00) >> 8)
	*pos++
	buffer[*pos] = byte((m.Frequency & 0xFF0000) >> 16)
	*pos++

	buffer[*pos] = byte(m.MaxDR&0x0F)<<4 | byte(m.MinDR&0x0F)
	*pos++
	return nil
}

func (m *MACPingSlotChannelReq) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.Frequency = uint32(buffer[*pos]) + uint32(buffer[*pos+1])<<8 + uint32(buffer[*pos+2])<<16
	*pos += 3
	m.MaxDR = (buffer[*pos] & 0xF0) >> 4
	m.MinDR = buffer[*pos] & 0X0F
	*pos++
	return nil
}

// MACPingSlotFreqAns is sent by the end-device to acknowledge a PingSlotChannelReq command
type MACPingSlotFreqAns struct {
	macBase
	DataRangeOK        bool
	ChannelFrequencyOK bool
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACPingSlotFreqAns) Length() int {
	return 2
}
func (m *MACPingSlotFreqAns) encode(buffer []byte, pos *int) error {
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

func (m *MACPingSlotFreqAns) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.DataRangeOK = buffer[*pos]&0x02 != 0
	m.ChannelFrequencyOK = buffer[*pos]&0x01 != 0
	*pos++
	return nil
}

// MACBeaconTimingReq is sent by the end-device to request the next timing and channel
type MACBeaconTimingReq struct {
	macBase
	// Empty payload
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACBeaconTimingReq) Length() int {
	return 1
}

func (m *MACBeaconTimingReq) encode(buffer []byte, pos *int) error {
	return encodeID(m, buffer, pos)
}

func (m *MACBeaconTimingReq) decode(buffer []byte, pos *int) error {
	return decodeID(m, buffer, pos)
}

// MACBeaconTimingAns is sent by the network server to the end-device in response to a BeaconTimingReq command
type MACBeaconTimingAns struct {
	macBase
	Delay   uint16 // Delay until next beacon [14.5]
	Channel uint8  // Index of channel where the next beacon will be broadcasted
}

// Length returns the length of the MAC command when encoded into a byte buffer
func (m *MACBeaconTimingAns) Length() int {
	return 4
}

func (m *MACBeaconTimingAns) encode(buffer []byte, pos *int) error {
	if err := encodeID(m, buffer, pos); err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(buffer[*pos:], m.Delay)
	*pos += 2
	buffer[*pos] = byte(m.Channel)
	*pos++
	return nil
}

func (m *MACBeaconTimingAns) decode(buffer []byte, pos *int) error {
	if err := decodeID(m, buffer, pos); err != nil {
		return err
	}
	m.Delay = binary.LittleEndian.Uint16(buffer[*pos:])
	*pos += 2
	m.Channel = buffer[*pos]
	*pos++
	return nil
}
