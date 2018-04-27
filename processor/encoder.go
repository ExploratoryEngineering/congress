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
	"time"

	"github.com/ExploratoryEngineering/congress/monitoring"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

// Encoder receives LoRaMessage data structures on a channel, encodes into a
// binary buffer and sends the buffer as a GatewayPacket instance on a new
// channel.
type Encoder struct {
	input   <-chan server.LoRaMessage
	output  chan<- server.GatewayPacket
	context *server.Context
}

func (e *Encoder) processMessage(packet server.LoRaMessage) {
	packet.FrameContext.GatewayContext.SectionTimer.Begin(monitoring.TimeEncoder)
	var buffer []byte
	var err error

	switch packet.Payload.MHDR.MType {

	case protocol.JoinRequest:
		logging.Warning("Unsupported encoding: JoinRequest (context=%v)", packet.FrameContext)

	case protocol.UnconfirmedDataUp:
		logging.Warning("Unsupported encoding: UnconfirmedDataUp (context=%v)", packet.FrameContext)

	case protocol.ConfirmedDataUp:
		logging.Warning("Unsupported encoding: ConfirmedDataUp (context=%v)", packet.FrameContext)

	case protocol.RFU:
		logging.Warning("Unsupported encoding: RFU(context=%v)", packet.FrameContext)

	case protocol.Proprietary:
		logging.Warning("Unsupported encoding: Proprietary message (context=%v)", packet.FrameContext)

	case protocol.JoinAccept:
		// Reset frame counter for both
		packet.FrameContext.Device.FCntDn = 0
		packet.FrameContext.Device.FCntUp = 0
		if err := e.context.Storage.Device.UpdateState(packet.FrameContext.Device); err != nil {
			logging.Warning("Unable to update frame counters for device with EUI %s: %v. Ignoring JoinRequest.", packet.FrameContext.Device.DeviceEUI, err)
			return
		}

		buffer, err = packet.Payload.EncodeJoinAccept(packet.FrameContext.Device.AppKey)
		if err != nil {
			logging.Warning("Unable to encode JoinAccept message for device with EUI %s (DevAddr=%s): %v",
				packet.FrameContext.Device.DeviceEUI,
				packet.FrameContext.Device.DevAddr,
				err)
			return
		}
		packet.FrameContext.GatewayContext.Radio.RX1Delay = 5
		packet.FrameContext.GatewayContext.Deadline = 5

	default:
		packet.Payload.MACPayload.FHDR.FCnt = packet.FrameContext.Device.FCntDn
		buffer, err = packet.Payload.EncodeMessage(packet.FrameContext.Device.NwkSKey, packet.FrameContext.Device.AppSKey)
		if err != nil {
			logging.Error("Unable to encode message for device with EUI %s: %v. (DevAddr=%s)",
				packet.FrameContext.Device.DeviceEUI,
				err,
				packet.FrameContext.Device.DevAddr)
			return
		}

		// Update the sent time for the message
		sentTime := time.Now().Unix()
		if err := e.context.Storage.DeviceData.UpdateDownstream(packet.FrameContext.Device.DeviceEUI, sentTime, 0); err != nil && err != storage.ErrNotFound {
			logging.Warning("Unable to update downstream message for device %s: %v", packet.FrameContext.Device.DeviceEUI, err)
		}

		// Increase the frame counter after the message is sent. New devices will get 0,1,2...
		packet.FrameContext.Device.FCntDn++
		if err := e.context.Storage.Device.UpdateState(packet.FrameContext.Device); err != nil {
			logging.Error("Unable to update frame counter for downstream message to device with EUI %s: %v",
				packet.FrameContext.Device.DeviceEUI,
				err)
		}
		packet.FrameContext.GatewayContext.Radio.RX1Delay = 1
		packet.FrameContext.GatewayContext.Deadline = 1
	}

	if len(buffer) == 0 {
		return
	}

	packet.FrameContext.GatewayContext.SectionTimer.End()
	// Copy relevant data to the outgoing packet.
	monitoring.Stopwatch(monitoring.EncoderChannelOut, func() {
		e.output <- server.GatewayPacket{
			RawMessage:   buffer,
			Radio:        packet.FrameContext.GatewayContext.Radio,
			Gateway:      packet.FrameContext.GatewayContext.Gateway,
			SectionTimer: packet.FrameContext.GatewayContext.SectionTimer,
			OutTimer:     packet.FrameContext.GatewayContext.OutTimer,
			ReceivedAt:   packet.FrameContext.GatewayContext.ReceivedAt,
			Deadline:     packet.FrameContext.GatewayContext.Deadline,
		}
	})
	monitoring.Encoder.Increment()
	monitoring.GetGatewayCounters(packet.FrameContext.GatewayContext.Gateway.GatewayEUI).MessagesOut.Increment()
	monitoring.GetAppCounters(packet.FrameContext.Application.AppEUI).MessagesOut.Increment()
}

// Start starts the Encoder instance. It will terminate when the input channel
// is closed. The output channel is closed when the method stops. The input channel
// receives messages due to be sent to gateways a short time before the messages
// must be sent from the gateway.
func (e *Encoder) Start() {
	for packet := range e.input {
		go e.processMessage(packet)

	}
	logging.Debug("Input channel for Encoder closed. Terminating")
}

// NewEncoder creates a new Encoder instance.
func NewEncoder(context *server.Context, input <-chan server.LoRaMessage, output chan<- server.GatewayPacket) *Encoder {
	return &Encoder{
		context: context,
		input:   input,
		output:  output,
	}
}
