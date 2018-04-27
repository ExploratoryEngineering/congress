package main

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
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/logging"
)

// EmulatedDevice emulates a device
type EmulatedDevice struct {
	keys              DeviceKeys
	sentMessageCount  int    // Number of messages currently sent
	FrameCounterUp    uint16 // Current frame counter
	FrameCounterDown  uint16
	duplicateMessages *TheRandomizer
	publisher         *EventRouter
	Config            Params
	incomingMessages  <-chan GWMessage
	outgoingMessages  chan string // Messages to be sent to the packet forwarder
	nonces            []uint16
	ReceivedMessages  []IncomingMessage
	Ack               bool // ack flag for upstream messages
}

// IncomingMessage is messages recevied from the server
type IncomingMessage struct {
	MessageType protocol.MType
	Payload     string
}

// NewEmulatedDevice creates a new OTAA device
func NewEmulatedDevice(config Params, keys DeviceKeys, outgoing chan string, publisher *EventRouter) *EmulatedDevice {
	return &EmulatedDevice{
		keys:              keys,
		sentMessageCount:  0,
		FrameCounterUp:    0,
		FrameCounterDown:  0,
		duplicateMessages: NewRandomizer(config.DuplicateMessages),
		publisher:         publisher,
		Config:            config,
		outgoingMessages:  outgoing,
		nonces:            make([]uint16, 0),
		ReceivedMessages:  make([]IncomingMessage, 0),
	}
}

// Join starts joining.
func (d *EmulatedDevice) Join(maxAttempts int) error {
	attempt := 0
	// DevAddr is set to 0 at this point in time
	d.incomingMessages = d.publisher.Subscribe(protocol.DevAddrFromUint32(0))
	defer d.publisher.Unsubscribe(d.incomingMessages)

	for {
		lastNonce := d.sendJoinRequest()
		// Keep reading responses until one matches or the request
		// times out.
		startTime := time.Now()
		waitTime := time.Now().Sub(startTime)
		for waitTime < time.Second*7 {
			select {
			case joinResponse := <-d.incomingMessages:
				if d.validJoinResponse(joinResponse, lastNonce) {
					logging.Info("Device %s has joined", d.keys.DevEUI)
					// Success - got message
					return nil
				}
			default:
				// Do nothing here.
			}
			time.Sleep(1 * time.Millisecond)
			waitTime = time.Now().Sub(startTime)
		}
		attempt++
		logging.Info("Device %s have used %d of %d join attempts", d.keys.DevEUI, attempt, maxAttempts)
		if attempt >= maxAttempts {
			return errors.New("no JoinAccept received from server")
		}
	}
}

// Generate a JoinRequest and send it to the packet forwarder/server
func (d *EmulatedDevice) sendJoinRequest() uint16 {
	p := protocol.NewPHYPayload(protocol.JoinRequest)
	lastNonce := d.makeNonce()
	p.JoinRequestPayload.AppEUI = d.keys.AppEUI
	p.JoinRequestPayload.DevEUI = d.keys.DevEUI
	p.JoinRequestPayload.DevNonce = lastNonce
	joinRequestBuf, err := p.EncodeJoinRequest(d.keys.AppKey)
	if err != nil {
		logging.Warning("Unable to encode JoinRequest for device %s: %v", d.keys.DevEUI, err)
		return lastNonce
	}
	logging.Info("Device %s sending JoinRequest", d.keys.DevEUI)
	d.outgoingMessages <- base64.StdEncoding.EncodeToString(joinRequestBuf)
	return lastNonce
}

func (d *EmulatedDevice) validJoinResponse(msg GWMessage, lastNonce uint16) bool {
	if msg.PHYPayload.MHDR.MType == protocol.JoinAccept {
		// Check the MIC to see if this a message for me. This is getting hairy since the payload is
		// encoded, mic is calculated and then the whole thing is encrypted.

		if err := msg.PHYPayload.DecodeJoinAccept(d.keys.AppKey, msg.Buffer); err != nil {
			if err != protocol.ErrInvalidMIC {
				logging.Warning("Couldn't decode the JoinAccept message: %v", err)
			}
			return false
		}
		// Got it - this is the message we're waiting for
		d.keys.DevAddr = msg.PHYPayload.JoinAcceptPayload.DevAddr
		d.keys.AppSKey, _ = protocol.AppSKeyFromNonces(d.keys.AppKey, msg.PHYPayload.JoinAcceptPayload.AppNonce, d.Config.NetID, lastNonce)
		d.keys.NwkSKey, _ = protocol.NwkSKeyFromNonces(d.keys.AppKey, msg.PHYPayload.JoinAcceptPayload.AppNonce, d.Config.NetID, lastNonce)
		d.FrameCounterDown = 0
		d.FrameCounterUp = 0
		return true
	}
	logging.Info("Device %s didn't get a JoinAccept but %s", d.keys.DevEUI, msg.PHYPayload.MHDR.MType)
	return false
}

// SendMessageWithGenerator sends a new message using the given message generator
func (d *EmulatedDevice) SendMessageWithGenerator(generator MessageGenerator) error {
	msg, mType := generator.Generate(d.keys, d.FrameCounterUp)
	d.incomingMessages = d.publisher.Subscribe(d.keys.DevAddr)
	defer d.publisher.Unsubscribe(d.incomingMessages)

	// Make a random message
	d.FrameCounterUp++
	d.outgoingMessages <- msg
	d.duplicateMessages.Maybe(func() {
		d.outgoingMessages <- msg
	})

	// Emulate receive window
	select {
	case response := <-d.incomingMessages:
		// If there's a payload grab it and add it to the list of received messages
		if len(response.PHYPayload.MACPayload.FRMPayload) > 0 {
			msg := IncomingMessage{response.PHYPayload.MHDR.MType, hex.EncodeToString(response.PHYPayload.MACPayload.FRMPayload)}
			d.ReceivedMessages = append(d.ReceivedMessages, msg)
		}
		if mType == protocol.ConfirmedDataUp && !response.PHYPayload.MACPayload.FHDR.FCtrl.ACK {
			return fmt.Errorf("Did get message but not an ACK after message for device %s", d.keys.DevAddr)
		}
	case <-time.After(3 * time.Second):
		if mType == protocol.ConfirmedDataUp {
			return errors.New("Didn't get an ack after 3 seconds")
		}
	}

	return nil
}

// SendMessageWithPayload sends a message with the specified type and payload
func (d *EmulatedDevice) SendMessageWithPayload(mtype protocol.MType, payload []byte) error {
	d.incomingMessages = d.publisher.Subscribe(d.keys.DevAddr)
	defer d.publisher.Unsubscribe(d.incomingMessages)

	message := protocol.NewPHYPayload(mtype)
	message.MACPayload.FHDR.DevAddr = d.keys.DevAddr
	message.MACPayload.FPort = uint8(rand.Intn(223) + 1)
	message.MACPayload.FHDR.FCnt = d.FrameCounterUp
	d.FrameCounterUp++

	message.MACPayload.FRMPayload = payload
	message.MACPayload.FHDR.FCtrl.ACK = d.Ack

	buffer, err := message.EncodeMessage(d.keys.NwkSKey, d.keys.AppSKey)
	if err != nil {
		return err
	}

	d.outgoingMessages <- base64.StdEncoding.EncodeToString(buffer)

	select {
	case response := <-d.incomingMessages:
		if len(response.PHYPayload.MACPayload.FRMPayload) > 0 {
			logging.Info("Received message for device %s", d.keys.DevEUI)
			// Decrypt the message

			plaintext := decryptPayload(
				d.keys.DevAddr,
				response.PHYPayload.MACPayload.FHDR.FCnt,
				d.keys.AppSKey,
				response.PHYPayload.MACPayload.FRMPayload)
			msg := IncomingMessage{response.PHYPayload.MHDR.MType, hex.EncodeToString(plaintext)}
			d.ReceivedMessages = append(d.ReceivedMessages, msg)
		}
		if mtype == protocol.ConfirmedDataUp && !response.PHYPayload.MACPayload.FHDR.FCtrl.ACK {
			return fmt.Errorf("Did get message but not an ACK after message for device %s", d.keys.DevAddr)
		}
		d.FrameCounterDown = response.PHYPayload.MACPayload.FHDR.FCnt
	case <-time.After(3 * time.Second):
		if mtype == protocol.ConfirmedDataUp {
			return errors.New("Didn't get an ack after 3 seconds")
		}
	}
	return nil
}

func (d *EmulatedDevice) makeNonce() uint16 {
	lastNonce := uint16(rand.Int31n(0xFFFF))
	d.nonces = append(d.nonces, lastNonce)
	return lastNonce
}

// Since the server side only implements the server side AES encryption it consistently decrypts
// the devices use Encrypt for both encryption and decryption. Just to confuse everyone a bit more.
func decryptPayload(devAddr protocol.DevAddr, fCntDown uint16, key protocol.AESKey, buf []byte) []byte {
	k := int(math.Ceil(float64(len(buf)) / 16))

	var S []byte
	for i := 0; i < k; i++ {
		A := make([]byte, 16)
		A[0] = 0x01

		A[5] = 1 // Always downlink
		binary.LittleEndian.PutUint32(A[6:], devAddr.ToUint32())
		binary.LittleEndian.PutUint32(A[10:], uint32(fCntDown))

		A[15] = byte(i + 1)

		Si := make([]byte, 16)
		block, _ := aes.NewCipher(key.Key[0:])
		block.Encrypt(Si, A)

		S = append(S, Si...)

	}

	plainText := make([]byte, len(buf))
	for i := 0; i < len(buf); i++ {
		plainText[i] = buf[i] ^ S[i]
	}

	return plainText
}
