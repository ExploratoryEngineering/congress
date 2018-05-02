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
// MACPayload is the payload in the MAC frame [4.3]
type MACPayload struct {
	FHDR        FHDR          // [4.3.1]
	FPort       uint8         // [4.3.2]
	FRMPayload  []byte        // [4.3.3]
	MACCommands MACCommandSet // These are the MAC commands in the payload (if appliccable)
}

// NewMACPayload creates a new MACPayload instance
func NewMACPayload(message MType) MACPayload {
	return MACPayload{
		MACCommands: NewMACCommandSet(message, maxPayloadSize),
	}
}

// BUG(stalehd): The payload size should be determined via the frequency plan, not as a constant.
const maxPayloadSize int = 255

func (m *MACPayload) encode(buffer []byte, count *int) error {
	if count == nil {
		return ErrNilError
	}
	if len(buffer) < *count {
		return ErrBufferTruncated
	}
	if len(m.FRMPayload) == 0 {
		// Port might be omitted but set it to 0
		m.FPort = 0
	}
	if m.FPort > 223 {
		return ErrParameterOutOfRange
	}
	if m.FPort == 0 && len(m.FRMPayload) > 0 {
		return ErrParameterOutOfRange
	}
	if len(m.FRMPayload) == 0 && m.MACCommands.Size() > 0 {
		// FPort shall be 0 if there's MAC commands in the payload
		m.FPort = 0
		buffer[*count] = m.FPort
		*count++
		return m.MACCommands.encode(buffer, count)
	}
	if len(m.FRMPayload) > 0 {
		buffer[*count] = byte(m.FPort)
		*count++
		copy(buffer[*count:*count+len(m.FRMPayload)], m.FRMPayload)
		*count += len(m.FRMPayload)
	}
	return nil
}

// decodeMACPayload extracts Frame header (FHDR), Port (FPort) and Frame Payload (FRMPayload)
func (m *MACPayload) decode(payload []byte, pos *int) error {
	if err := m.FHDR.decode(payload, pos); err != nil {
		return err
	}
	payloadLength := len(payload) - *pos - 4 /* MIC */
	if payloadLength == 1 || payloadLength < 0 {
		// payload must include port so port + payload can't be 1
		return ErrBufferTruncated
	}
	if payloadLength == 0 {
		// Nothing more to do
		return nil
	}

	m.FPort = payload[*pos]
	*pos++
	if m.FPort == 0 {
		m.MACCommands = NewMACCommandSet(m.MACCommands.Message(), payloadLength)
		if err := m.MACCommands.decode(payload, pos); err != nil {
			if err == errUnknownMAC {
				return nil
			}
			return err
		}
	}

	m.FRMPayload = payload[*pos : *pos+payloadLength-1]

	return nil
}
