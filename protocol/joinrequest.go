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

// JoinRequestPayload is the payload sent by the device in a JoinRequest
// message [6.2.4]. The message is not encrypted.
type JoinRequestPayload struct {
	AppEUI   EUI
	DevEUI   EUI
	DevNonce uint16
}

// Decode JoinRequest payload from a byte buffer.
func (j *JoinRequestPayload) decode(buffer []byte, pos *int) error {
	if buffer == nil || pos == nil {
		return ErrNilError
	}
	if len(buffer) < (*pos + 18) {
		return ErrBufferTruncated
	}
	j.AppEUI = EUIFromUint64(binary.LittleEndian.Uint64(buffer[*pos:]))
	*pos += 8
	j.DevEUI = EUIFromUint64(binary.LittleEndian.Uint64(buffer[*pos:]))
	*pos += 8

	// These should be big endian. Because keys.
	j.DevNonce = binary.BigEndian.Uint16(buffer[*pos:])
	*pos += 2
	return nil
}

// Encode the JoinRequest into a buffer
func (j *JoinRequestPayload) encode(buffer []byte, pos *int) error {
	if buffer == nil || pos == nil {
		return ErrNilError
	}
	if len(buffer) < (*pos + 18) {
		return ErrBufferTruncated
	}

	binary.LittleEndian.PutUint64(buffer[*pos:], j.AppEUI.ToUint64())
	*pos += 8
	binary.LittleEndian.PutUint64(buffer[*pos:], j.DevEUI.ToUint64())
	*pos += 8

	// These should be big endian. Because keys.
	binary.BigEndian.PutUint16(buffer[*pos:], j.DevNonce)
	*pos += 2
	return nil
}
