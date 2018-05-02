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
	"crypto/rand"
)

// JoinAcceptPayload is the payload sent by the network server to an
// end-device in response to a JoinRequest message. The message is
// encrypted with the (AES) application key. [6.2.5]
type JoinAcceptPayload struct {
	AppNonce   [3]byte
	NetID      uint32
	DevAddr    DevAddr
	DLSettings DLSettings
	RxDelay    byte
	CFList     CFList
}

// GenerateAppNonce generates a new AppNonce for an application. [6.2.5]
func (j *JoinAcceptPayload) GenerateAppNonce() ([3]byte, error) {
	var ret [3]byte
	if _, err := rand.Read(ret[:]); err != nil {
		return ret, err
	}
	return ret, nil
}

// Encode payload into byte buffer.
func (j *JoinAcceptPayload) encode(buffer []byte, pos *int) error {
	if buffer == nil || pos == nil {
		return ErrNilError
	}
	if len(buffer) < (*pos + 12) {
		return ErrBufferTruncated
	}
	copy(buffer[*pos:], j.AppNonce[:])
	*pos += 3
	buffer[*pos+0] = byte((j.NetID >> 16) & 0xFF)
	buffer[*pos+1] = byte((j.NetID >> 8) & 0xFF)
	buffer[*pos+2] = byte((j.NetID >> 0) & 0xFF)
	*pos += 3
	if err := j.DevAddr.encode(buffer, pos); err != nil {
		return err
	}
	if err := j.DLSettings.encode(buffer, pos); err != nil {
		return err
	}
	buffer[*pos] = j.RxDelay
	*pos++
	return nil
}

func (j *JoinAcceptPayload) decode(buffer []byte, pos *int) error {
	if buffer == nil || pos == nil {
		return ErrNilError
	}
	if len(buffer) < (*pos + 6) {
		return ErrBufferTruncated
	}
	copy(j.AppNonce[:], buffer[*pos:*pos+3])
	*pos += 3
	j.NetID = uint32(buffer[*pos+0])<<16 + uint32(buffer[*pos+1])<<8 + uint32(buffer[*pos+2])
	*pos += 3
	if err := j.DevAddr.decode(buffer, pos); err != nil {
		return err
	}
	if err := j.DLSettings.decode(buffer, pos); err != nil {
		return err
	}
	j.RxDelay = buffer[*pos]
	*pos++
	return nil
}
