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
	"crypto/rand"
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/band"
	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/logging"
)

var fo = server.NewFrameOutputBuffer()

var context = server.Context{
	FrameOutput: &fo,
}

var frameContext = server.FrameContext{
	GatewayContext: server.GatewayPacket{
		Radio: server.RadioContext{
			Band:      band.EU868{},
			DataRate:  "SF10BW125",
			Frequency: 868.1,
		},
	},
	Device: model.Device{
		DeviceEUI: protocol.EUIFromUint64(0x0102030405060708),
		DevAddr:   protocol.DevAddrFromUint32(0x01020304),
	},
}

func TestSchedulerChannels(t *testing.T) {
	input := make(chan server.LoRaMessage)

	scheduler := NewScheduler(&context, input)

	go scheduler.Start()

	close(input)
	select {
	case _, ok := <-scheduler.Output():
		if ok {
			t.Fatal("Expected output channel to be closed")
		}
	}
}

var addr uint32

func makeRandomMessage() server.LoRaMessage {
	addr++
	payload := protocol.NewPHYPayload(protocol.UnconfirmedDataDown)
	payload.MACPayload.FHDR.DevAddr = frameContext.Device.DevAddr
	payload.MACPayload.FRMPayload = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	return server.LoRaMessage{
		Payload:      payload,
		FrameContext: frameContext,
	}
}
func TestScheduler(t *testing.T) {
	input := make(chan server.LoRaMessage)

	scheduler := NewScheduler(&context, input)

	// Pull a single message on the input channel, check if something
	// comes out on the other side within rxwindow - margin
	messageToSend := makeRandomMessage()

	// Populate the device output with the same data from the input channel
	context.FrameOutput.SetPayload(
		messageToSend.FrameContext.Device.DeviceEUI,
		messageToSend.Payload.MACPayload.FRMPayload,
		messageToSend.Payload.MACPayload.FPort, false)

	go scheduler.Start()

	input <- messageToSend

	select {
	case <-scheduler.Output():
	// OK
	case <-time.After(time.Second * 1):
		t.Fatal("Did not get a message within the expected time frame")
	}
}

func newFrameContext(counter uint32) server.FrameContext {
	return server.FrameContext{
		GatewayContext: server.GatewayPacket{
			Radio: server.RadioContext{
				Band:      band.EU868{},
				DataRate:  "SF10BW125",
				Frequency: 868.1,
			},
		},
		Device: model.Device{
			DeviceEUI: protocol.EUIFromUint64(uint64(counter)),
			DevAddr:   protocol.DevAddr{NwkID: 1, NwkAddr: counter},
		},
	}
}

func TestMultiMessages(t *testing.T) {
	logging.SetLogLevel(logging.DebugLevel)
	input := make(chan server.LoRaMessage)

	scheduler := NewScheduler(&context, input)

	go scheduler.Start()

	delays := make([]byte, 255)
	if _, err := rand.Read(delays); err != nil {
		t.Fatal("Couldn't get random numbers. Exiting.")
	}
	for i, v := range delays {
		go func(delay time.Duration, num uint32) {
			delay = delay / 4
			<-time.After(delay * time.Millisecond)

			// Make sure there's no duplicates wrt the DevAddr struct
			msg := makeRandomMessage()
			msg.FrameContext = newFrameContext(num)
			msg.Payload.MACPayload.FHDR.DevAddr = msg.FrameContext.Device.DevAddr
			context.FrameOutput.SetPayload(
				msg.FrameContext.Device.DeviceEUI,
				msg.Payload.MACPayload.FRMPayload,
				msg.Payload.MACPayload.FPort, false)

			input <- msg
		}(time.Duration(v), uint32(i))
	}

	received := 0
	for {
		select {
		case <-scheduler.Output():
			received++
			if received == len(delays) {
				return
			}
		case <-time.After(time.Second * 1):
			t.Fatalf("Didn't receive all of the messages within 1 second. Got %d of %d", received, len(delays))
		}
	}
}

// Ensure that scheduler only emits one message when duplicates are scheduled.
func TestSchedulerDuplicate(t *testing.T) {
	input := make(chan server.LoRaMessage)

	scheduler := NewScheduler(&context, input)

	messageToSend := makeRandomMessage()

	// Populate the device output with the same data from the input channel
	context.FrameOutput.SetPayload(
		messageToSend.FrameContext.Device.DeviceEUI,
		messageToSend.Payload.MACPayload.FRMPayload,
		messageToSend.Payload.MACPayload.FPort, false)

	go scheduler.Start()

	input <- messageToSend
	// Sleep some, then send the message again. It should be skipped
	<-time.After(time.Millisecond * 100)
	input <- messageToSend

	select {
	case <-scheduler.Output():
	// OK
	case <-time.After(time.Second * 1):
		t.Fatal("Did not get a message within the expected time frame")
	}

	// Doing the same a 2nd time should time out
	select {
	case <-scheduler.Output():
		t.Fatal("Got message but didn't expect it to be so")
	case <-time.After(time.Millisecond * 500):
		// OK
	}
}
