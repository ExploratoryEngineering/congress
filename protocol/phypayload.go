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
	"crypto/aes"
	"encoding/binary"
	"errors"
	"math"
)

// PHYPayload is the payload in the PHY frame
type PHYPayload struct {
	MHDR               MHDR       // [4.2]
	MACPayload         MACPayload // [4.3]
	JoinRequestPayload JoinRequestPayload
	JoinAcceptPayload  JoinAcceptPayload
	MIC                uint32 // [4.4]
}

// MinimumMessageSize is the absolute minimum size for a LoRaWAN message. Messages
// shorter than this will be rejected outright
const MinimumMessageSize = 12

// NewPHYPayload creates a new PHYPayload instance with the specified direction
func NewPHYPayload(messageType MType) PHYPayload {
	return PHYPayload{
		MHDR: MHDR{MType: messageType, MajorVersion: LoRaWANR1},
		MACPayload: MACPayload{
			FHDR: FHDR{
				FOpts: NewFOptsSet(messageType),
			},
			FRMPayload:  make([]byte, 0),
			MACCommands: NewMACCommandSet(messageType, 222),
		},
	}
}

// UnmarshalBinary extracts MAC Header, MAC Payload and Message Integrity Code (MIC)
func (p *PHYPayload) UnmarshalBinary(data []byte) error {
	// The minimum size of a LoRaWAN buffer is 12 bytes (MHDR:1, FHDR:4+1, FCnt:2, MIC: 4)
	if len(data) < MinimumMessageSize {
		return ErrBufferTruncated
	}
	var err error
	pos := 0
	if err = p.MHDR.decode(data, &pos); err != nil {
		return err
	}
	p.MIC = binary.LittleEndian.Uint32(data[len(data)-4:])

	if p.MHDR.MType == ConfirmedDataUp ||
		p.MHDR.MType == UnconfirmedDataUp ||
		p.MHDR.MType == UnconfirmedDataDown ||
		p.MHDR.MType == ConfirmedDataDown {
		p.MACPayload.MACCommands = NewMACCommandSet(p.MHDR.MType, maxPayloadSize)
		p.MACPayload.FHDR.FOpts = NewFOptsSet(p.MHDR.MType)
		return p.MACPayload.decode(data, &pos)
	}

	if p.MHDR.MType == JoinRequest {
		pos := 1
		return p.JoinRequestPayload.decode(data, &pos)
	}
	if p.MHDR.MType == JoinAccept {
		return p.JoinAcceptPayload.decode(data, &pos)
	}
	return ErrInvalidMessageType
}

// ErrInvalidMIC is returned when the JoinAccept MIC is invalid
var ErrInvalidMIC = errors.New("invalid MIC")

// DecodeJoinAccept decodes the JoinAccept payload from message. The MHDR is
// assumed to be decoded (via UnmarshalBinary) before this method is called.
// The MIC will be updated and verified against the calculated MIC. If the
// MIC is incorrect it will return ErrInvalidMIC
func (p *PHYPayload) DecodeJoinAccept(aesKey AESKey, buffer []byte) error {
	if p.MHDR.MType != JoinAccept {
		return ErrInvalidMessageType
	}
	cipher, err := aes.NewCipher(aesKey.Key[:])
	if err != nil {
		return err
	}
	paddedLen := 16
	for paddedLen < (len(buffer) - 1) {
		paddedLen += 16
	}
	input := make([]byte, paddedLen)
	copy(input, buffer[1:])
	decrypted := make([]byte, paddedLen)

	cipher.Encrypt(decrypted, input)

	pos := 0
	p.JoinAcceptPayload.AppNonce[0] = decrypted[pos+0]
	p.JoinAcceptPayload.AppNonce[1] = decrypted[pos+1]
	p.JoinAcceptPayload.AppNonce[2] = decrypted[pos+2]
	pos += 3
	p.JoinAcceptPayload.NetID = uint32(decrypted[pos+0])<<16 +
		uint32(decrypted[pos+1])<<8 +
		uint32(decrypted[pos+2])

	pos += 3
	if err := p.JoinAcceptPayload.DevAddr.decode(decrypted, &pos); err != nil {
		return err
	}
	if err := p.JoinAcceptPayload.DLSettings.decode(decrypted, &pos); err != nil {
		return err
	}
	p.JoinAcceptPayload.RxDelay = decrypted[pos]

	// The decrypted buffer shouldn't include the MHDR (1 byte) or the MIC (4 byte)
	p.MIC, err = p.CalculateJoinAcceptMIC(aesKey, append(buffer[0:1], decrypted[0:len(buffer)-5]...))
	// Check if the MIC in the decrypted buffer matches the MIC we found
	bufferMIC := binary.LittleEndian.Uint32(decrypted[len(buffer)-5:])
	if bufferMIC != p.MIC {
		return ErrInvalidMIC
	}
	if err != nil {
		return err
	}
	return nil
}

// MarshalBinary marshals the struct into a byte buffer. Note that some fields might change
// as a result of this; most notably the FOptsLen field and the Port field. The payload must
// be encrypted at this point.
func (p *PHYPayload) MarshalBinary() ([]byte, error) {
	if p.MHDR.MType == JoinAccept || p.MHDR.MType == JoinRequest || p.MHDR.MType == Proprietary {
		return nil, ErrInvalidMessageType
	}
	count := 0
	buffer := make([]byte, 255)
	if err := p.MHDR.encode(buffer, &count); err != nil {
		return nil, err
	}

	if err := p.MACPayload.FHDR.encode(buffer, &count); err != nil {
		return nil, err
	}

	if err := p.MACPayload.encode(buffer, &count); err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint32(buffer[count:], p.MIC)
	count += 4

	return buffer[0:count], nil
}

// EncodeJoinAccept encodes a complete JoinAccept message, including
// MIC and encryption. The appKey parameter is the application key.
func (p *PHYPayload) EncodeJoinAccept(appKey AESKey) ([]byte, error) {
	if p.MHDR.MType != JoinAccept {
		return nil, ErrInvalidMessageType
	}
	var err error

	// Maximum size of JoinAccept is 3+3+4+1+1+16=28 bytes [6.2.5]
	buffer := make([]byte, 30)
	pos := 0
	if err = p.MHDR.encode(buffer, &pos); err != nil {
		return nil, err
	}
	if err = p.JoinAcceptPayload.encode(buffer, &pos); err != nil {
		return nil, err
	}

	if p.MIC, err = p.CalculateJoinAcceptMIC(appKey, buffer[0:pos]); err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint32(buffer[pos:], p.MIC)
	pos += 4

	// Now do the decryption see [6.2.5]
	cipher, err := aes.NewCipher(appKey.Key[:])
	if err != nil {
		return nil, err
	}
	ret := make([]byte, pos)
	ret[0] = buffer[0]                     // Use MHDR as is
	cipher.Decrypt(ret[1:], buffer[1:pos]) // and encrypt payload + mic

	return ret, nil
}

// EncodeJoinRequest encodes a JoinRequest message
func (p *PHYPayload) EncodeJoinRequest(appKey AESKey) ([]byte, error) {
	if p.MHDR.MType != JoinRequest {
		return nil, ErrInvalidMessageType
	}

	count := 0
	// JoinRequest is 1 (MHDR) + 18 (JoinRequest) + 5 (MIC) = 24 bytes
	buf := make([]byte, 24)

	if err := p.MHDR.encode(buf, &count); err != nil {
		return nil, err
	}
	if err := p.JoinRequestPayload.encode(buf, &count); err != nil {
		return nil, err
	}
	var err error
	if p.MIC, err = p.CalculateJoinRequestMIC(appKey, buf); err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint32(buf[count:], p.MIC)
	count += 4

	return buf, nil
}

// EncodeMessage encrypts and adds MIC for the message.
func (p *PHYPayload) EncodeMessage(nwkSKey AESKey, appSKey AESKey) ([]byte, error) {
	if err := p.encrypt(nwkSKey, appSKey); err != nil {
		return nil, err
	}
	return p.MarshalBinary()
}

// Encrypt encrypts message according to [4.3.3.1]
func (p *PHYPayload) encrypt(nwkSKey AESKey, appSKey AESKey) error {
	p.Decrypt(nwkSKey, appSKey)

	buf, err := p.MarshalBinary()
	if err != nil {
		return err
	}
	if len(buf) < 4 {
		return ErrBufferTruncated
	}
	p.MIC, err = p.CalculateMIC(nwkSKey, buf[0:len(buf)-4])
	return err
}

// Decrypt decrypts message according to [4.3.3.1]
func (p *PHYPayload) Decrypt(nwkSKey AESKey, appSKey AESKey) {
	var key *AESKey
	if p.MACPayload.FPort == 0 {
		key = &nwkSKey
	} else {
		key = &appSKey
	}

	k := int(math.Ceil(float64(len(p.MACPayload.FRMPayload)) / 16))

	var S []byte
	for i := 0; i < k; i++ {
		A := make([]byte, 16)
		A[0] = 0x01
		if !p.MHDR.MType.Uplink() {
			A[5] = 1
		}
		binary.LittleEndian.PutUint32(A[6:], p.MACPayload.FHDR.DevAddr.ToUint32())
		binary.LittleEndian.PutUint32(A[10:], uint32(p.MACPayload.FHDR.FCnt))

		A[15] = byte(i + 1)

		Si := make([]byte, 16)
		block, _ := aes.NewCipher(key.Key[0:])
		block.Encrypt(Si, A)

		S = append(S, Si...)

	}

	// FRMPayload can be cryptotext or plaintext. Decrypt/encrypt is symmetrical
	text := make([]byte, len(p.MACPayload.FRMPayload))
	for i := 0; i < len(p.MACPayload.FRMPayload); i++ {
		text[i] = p.MACPayload.FRMPayload[i] ^ S[i]
	}

	p.MACPayload.FRMPayload = text
}
