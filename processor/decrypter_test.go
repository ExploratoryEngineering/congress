package processor

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
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/pubsub"
)

func createEncryptedTestMessage() protocol.PHYPayload {
	message := protocol.NewPHYPayload(protocol.UnconfirmedDataUp)
	message.MACPayload.FHDR.DevAddr = protocol.DevAddr{
		NwkID:   0,
		NwkAddr: 0x1E672E6,
	}
	message.MACPayload.FHDR.FCtrl = protocol.FCtrl{
		ADR:       true,
		ADRACKReq: true,
		ACK:       false,
		FPending:  false,
		ClassB:    false,
		FOptsLen:  0,
	}
	message.MACPayload.FHDR.FCnt = 24
	message.MACPayload.FPort = 12
	message.MACPayload.FRMPayload = []byte{64, 238, 230, 130, 88, 184, 42, 7, 126, 23, 44, 234, 243, 24, 7, 221, 192, 181, 108, 89, 132, 44, 165, 42, 244}
	message.MIC = 0x22CBE65F

	return message
}

func TestDecrypterProcessing(t *testing.T) {

	s := NewStorageTestContext()
	router := pubsub.NewEventRouter(5)
	context := server.Context{Storage: &s, AppRouter: &router}

	input := make(chan server.LoRaMessage)

	decrypter := NewDecrypter(&context, input)

	go decrypter.Start()

	msg := createEncryptedTestMessage()
	d, _ := s.Device.GetByDevAddr(msg.MACPayload.FHDR.DevAddr)
	var msgOutput <-chan interface{}
	for device := range d {
		t.Logf("Found device: %T: with AppEUI %s", device, device.AppEUI)
		msgOutput = context.AppRouter.Subscribe(device.AppEUI)
	}

	byteMessage, err := msg.MarshalBinary()
	if err != nil {
		t.Fatal("MarshallBinary failed: ", err)
	}
	frameContext := server.FrameContext{
		GatewayContext: server.GatewayPacket{
			RawMessage: byteMessage,
		},
	}
	input <- server.LoRaMessage{Payload: createEncryptedTestMessage(), FrameContext: frameContext}

	outputNum := 0
	macOutputNum := 0
	timeoutNum := 0
	for i := 0; i < 3; i++ {
		select {
		case <-decrypter.Output():
			macOutputNum++
			if macOutputNum > 1 {
				t.Fatal("Expected just 1 read from mac output")
			}

		case <-msgOutput:
			outputNum++
			if outputNum > 1 {
				t.Fatal("Expected just 1 read from output")
			}
		// This is OK.
		case <-time.After(300 * time.Millisecond):
			timeoutNum++
			if timeoutNum > 1 {
				t.Fatalf("Expected just 1 timeout but got %d timeouts and %d MAC outputs %d payloads", timeoutNum, macOutputNum, outputNum)
			}
		}
	}
	close(input)
}
