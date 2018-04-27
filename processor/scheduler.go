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

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/logging"
)

// Scheduler is the process that schedules downlink frames. The sceduler reads
// from a command notifier channel and will schedule a frame to be sent to the
// device when it receives notification of an uplink. If the frame to be sent
// is empty it won't generate any output.
type Scheduler struct {
	notifier     <-chan server.LoRaMessage // Input channel; messages on this channel is received
	output       chan server.LoRaMessage   // Output channel; message will be sent when put on this channel
	scheduled    map[protocol.EUI]bool     // Map with devaddr for scheduled devices
	completed    chan protocol.EUI         // Channel for completed schedules
	context      *server.Context           // Server context
	fixedRxDelay time.Duration
}

// DefaultRXDelay is the default delay
const DefaultRXDelay time.Duration = 200 * time.Millisecond

// calculate the rx delay/wait time for the device with the specified DevAddr. Returned
// value is in us. (note that 1 second is incorrect)
// TODO(stalehd): Use device method for this.
func (s *Scheduler) calculateRxDelay(receivedMessage server.LoRaMessage) time.Duration {
	spentTime := time.Now().Sub(receivedMessage.FrameContext.GatewayContext.ReceivedAt)
	delay := s.fixedRxDelay - spentTime
	if delay < 0 {
		return 0
	}
	return delay
}

// SetRXDelay adjusts the RX delay. This is only for testing
func (s *Scheduler) SetRXDelay(newDelay time.Duration) {
	s.fixedRxDelay = newDelay
}

// Get the message to be sent from the device aggregator
func (s *Scheduler) buildMessageToSend(device model.Device, frameContext server.FrameContext) (server.LoRaMessage, error) {

	payload, err := s.context.FrameOutput.GetPHYPayloadForDevice(&device, &frameContext)
	if err != nil {
		logging.Debug("No data for device %s to send: %v", device.DeviceEUI, err)
	}
	return server.LoRaMessage{
		Payload:      payload,
		FrameContext: frameContext,
	}, err
}

// sendAt sends a message at a specified time
func (s *Scheduler) sendAt(delay time.Duration,
	device model.Device,
	output chan<- server.LoRaMessage,
	frameContext server.FrameContext,
	doneChannel chan protocol.EUI) {

	select {
	case <-time.After(delay):
		frameContext.GatewayContext.SectionTimer.Begin(monitoring.TimeSchedulerSend)
		payload, err := s.buildMessageToSend(device, frameContext)
		payload.FrameContext.GatewayContext.SectionTimer.End()
		// If there's an error there's no data to send.
		if err == nil {
			payload.FrameContext.GatewayContext.OutTimer.Begin(monitoring.TimeOutgoing)
			monitoring.Stopwatch(monitoring.SchedulerChannelOut, func() {
				output <- payload
			})
		}
		doneChannel <- device.DeviceEUI
		monitoring.SchedulerOut.Increment()
		switch payload.Payload.MHDR.MType {
		case protocol.ConfirmedDataDown:
			monitoring.LoRaConfirmedDown.Increment()
		case protocol.UnconfirmedDataDown:
			monitoring.LoRaUnconfirmedDown.Increment()
		}
	}
}

// Start launches the scheduler. When the notifier channel is closed it will stop
// and the output channel will be closed.
func (s *Scheduler) Start() {
	for {
		select {
		case message, ok := <-s.notifier:
			if !ok {
				close(s.output)
				return
			}
			message.FrameContext.GatewayContext.SectionTimer.Begin(monitoring.TimeSchedulerProcess)
			device := message.FrameContext.Device
			// Check if this message is already scheduled. If so - mark it as
			// a duplicate and skip it. Messages sent within the same n milliseconds
			// are assumed to be the same. This assumption is most likely wrong
			// but other parts of the pipeline will have to more extensive
			// duplicate/invalid data checks.
			if s.scheduled[device.DeviceEUI] {
				logging.Info("Found duplicate message from device with EUI %s", device.DeviceEUI)
				continue
			}

			// this isn't a duplicate. Add it
			s.scheduled[device.DeviceEUI] = true
			message.FrameContext.GatewayContext.SectionTimer.End()
			message.FrameContext.GatewayContext.InTimer.End()
			go s.sendAt(s.calculateRxDelay(message), device, s.output, message.FrameContext, s.completed)
			monitoring.SchedulerIn.Increment()

		case eui := <-s.completed:
			// Message has been sent. Remove it from the map
			delete(s.scheduled, eui)
		}
	}
}

// Output returns the output channel for the scheduler. A new message is sent
// on the channel whenever it is ready to be sent to a device.
func (s *Scheduler) Output() <-chan server.LoRaMessage {
	return s.output
}

// NewScheduler creates a new scheduler.
func NewScheduler(context *server.Context, commandNotifier <-chan server.LoRaMessage) *Scheduler {
	return &Scheduler{
		notifier:     commandNotifier,
		output:       make(chan server.LoRaMessage),
		context:      context,
		completed:    make(chan protocol.EUI),
		scheduled:    make(map[protocol.EUI]bool),
		fixedRxDelay: DefaultRXDelay,
	}
}
