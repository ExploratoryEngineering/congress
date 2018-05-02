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

// FHDR is the frame header [4.3.1]
type FHDR struct {
	DevAddr DevAddr       // [6.1.1]
	FCtrl   FCtrl         // [4.3.1]
	FCnt    uint16        // [4.3.1.5]
	FOpts   MACCommandSet // MAC Commands in the FOpts structure
}

// decodeFHDR extracts Device Address, Frame Control octet, Frame Counter, Frame Options from Frame Header
func (f *FHDR) decode(octets []byte, pos *int) error {
	if err := f.DevAddr.decode(octets, pos); err != nil {
		return err
	}

	if err := f.FCtrl.decode(octets, pos); err != nil {
		return err
	}

	if len(octets) < *pos+2 {
		return ErrBufferTruncated
	}
	f.FCnt = binary.LittleEndian.Uint16(octets[*pos : *pos+2])
	*pos += 2

	if f.FCtrl.FOptsLen > 0 {
		f.FOpts = NewMACCommandSet(f.FOpts.Message(), int(f.FCtrl.FOptsLen))
		if err := f.FOpts.decode(octets, pos); err != nil {
			if err == errUnknownMAC {
				// Found an unknown MAC command. Skip forward the number of missing bytes
				*pos += (int(f.FCtrl.FOptsLen) - f.FOpts.EncodedLength())
				return nil
			}
			return err
		}
	}
	return nil
}

func (f *FHDR) encode(buffer []byte, count *int) error {
	if count == nil {
		return ErrNilError
	}
	if len(buffer) < (*count + 5) {
		return ErrBufferTruncated
	}
	var devaddr uint32
	devaddr = uint32(f.DevAddr.NwkID)<<25 | uint32(f.DevAddr.NwkAddr&0x1FFFFFF)

	binary.LittleEndian.PutUint32(buffer[*count:], devaddr)

	*count += 4

	f.FCtrl.FOptsLen = uint8(f.FOpts.EncodedLength())
	if err := f.FCtrl.encode(buffer, count); err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(buffer[*count:*count+2], f.FCnt)
	*count += 2

	if err := f.FOpts.encode(buffer, count); err != nil {
		return err
	}

	return nil
}
