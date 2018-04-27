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
	"github.com/ExploratoryEngineering/congress/monitoring"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/logging"
)

// MACProcessor is the process responsible for processing the MAC commands.
type MACProcessor struct {
	input    <-chan server.LoRaMessage // Input from decoder; receives decoded, deduped and valid frame
	notifier chan server.LoRaMessage   // Notifier output; notifies scheduler about new RX
	context  *server.Context           // Server context
}

func processMACCommand(cmd protocol.MACCommand) {
	switch cmd.ID() {
	case protocol.LinkCheckReq:
		// Initiated by the end device
		logging.Warning("LinkCheckReq support not implemented")
	case protocol.LinkADRAns:
		logging.Warning("LinkADRAns support not implemented")
	case protocol.DutyCycleAns:
		logging.Warning("DutyCycleAns support not implemented")
	case protocol.RXParamSetupAns:
		logging.Warning("RXParamSetupAns support not implemented")
	case protocol.DevStatusAns:
		logging.Warning("DevStatusAns support not implemented")
	case protocol.NewChannelAns:
		logging.Warning("NewChannelAns support not implemented")
	case protocol.RXTimingSetupAns:
		logging.Warning("RXTimingSetupAns support not implemented")
	case protocol.PingSlotInfoReq:
		// Initiated by the end device
		logging.Warning("PingSlotInfoReq support not implemented")
	case protocol.BeaconTimingReq:
		// Initiated by the end device
		logging.Warning("BeaconTimingReq support not implemented")
	case protocol.PingSlotFreqAns:
		logging.Warning("PingSlotFreqAns support not implemented")
	case protocol.BeaconFreqAns:
		logging.Warning("BeaconFreqAns support not implemented")
	default:
		logging.Warning("Unknown MAC command: %d", cmd.ID())
	}
}

// Start launches the MAC processor. When the input channel is closed the
// method will stop and the notifier channel will be closed.
func (m *MACProcessor) Start() {
	for v := range m.input {
		go func(val server.LoRaMessage) {
			val.FrameContext.GatewayContext.SectionTimer.Begin(monitoring.TimeMACProcessor)
			for _, cmd := range val.Payload.MACPayload.MACCommands.List() {
				processMACCommand(cmd)
			}
			for _, cmd := range val.Payload.MACPayload.FHDR.FOpts.List() {
				processMACCommand(cmd)
			}
			val.FrameContext.GatewayContext.SectionTimer.End()
			monitoring.Stopwatch(monitoring.MACProcessorChannelOut, func() {
				m.notifier <- val
			})
			monitoring.MACProcessor.Increment()
		}(v)
	}
	logging.Debug("Input channel for MAC processor closed. Terminating")
	close(m.notifier)
}

// CommandNotifier returns the output channel for the MAC processor. A new
// message is sent on the channel every time the MAC processing step is
// completed, even if there's no new MAC commands that must be sent.
func (m *MACProcessor) CommandNotifier() <-chan server.LoRaMessage {
	return m.notifier
}

// NewMACProcessor creates a new MAC processor instance.
func NewMACProcessor(context *server.Context, input <-chan server.LoRaMessage) *MACProcessor {
	return &MACProcessor{
		context:  context,
		input:    input,
		notifier: make(chan server.LoRaMessage),
	}
}
