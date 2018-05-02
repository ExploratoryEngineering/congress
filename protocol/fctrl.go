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
// FCtrl contains frame control bits [4.3.1]
type FCtrl struct {
	ADR       bool  // [4.3.1.1]
	ADRACKReq bool  // [4.3.1.1]
	ACK       bool  // [4.3.1.2]
	FPending  bool  // [4.3.1.4] Is interpreted as RFU in uplink frames
	ClassB    bool  // [10] uplink frames have this bit set
	FOptsLen  uint8 // [4.3.1.6]
}

// MaxFOptsLen is the maximum length for the FOpts field.
const MaxFOptsLen int = 15

// decode FCtrl structure.
func (f *FCtrl) decode(buffer []byte, pos *int) error {
	if pos == nil {
		return ErrNilError
	}
	if len(buffer) <= *pos {
		return ErrParameterOutOfRange
	}
	f.ADR = buffer[*pos]&0x80 != 0
	f.ADRACKReq = buffer[*pos]&0x40 != 0
	f.ACK = buffer[*pos]&0x20 != 0
	// FPending is interpreted as RFU in downlink frames
	f.FPending = buffer[*pos]&0x10 != 0
	f.FOptsLen = buffer[*pos] & 0xF
	f.ClassB = buffer[*pos]&0x10 != 0
	*pos++
	return nil
}

func (f *FCtrl) encode(buffer []byte, count *int) error {
	if count == nil {
		return ErrNilError
	}
	if len(buffer) < *count {
		return ErrBufferTruncated
	}
	if f.FOptsLen > 0xF {
		return ErrParameterOutOfRange
	}
	buffer[*count] = byte(f.FOptsLen & 0xF)
	if f.ADR {
		buffer[*count] |= byte(1 << 7)
	}
	if f.ADRACKReq {
		buffer[*count] |= byte(1 << 6)
	}
	if f.ACK {
		buffer[*count] |= byte(1 << 5)
	}
	// Encode even if this might be unused. If this is a class A device
	// and this is an uplink frame this field is RFU.
	// For Class B devices the field signals a Class B device.
	// Uplink frames are the most relevant for our use (at least as long as
	// clients aren't running Go :) )
	if f.FPending || f.ClassB {
		buffer[*count] |= byte(1 << 4)
	}
	*count++
	return nil
}
