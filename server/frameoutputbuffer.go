package server

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
	"errors"
	"fmt"
	"sync"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/logging"
)

// frameOutput holds the aggregated output for the device that will fit into
// one packet, ie payload + up to 16 bytes of piggybacked MAC commands or just
// MAC commands up to the maximum payload size.
type frameOutput struct {
	MType             protocol.MType
	ADR               bool                   // ADR enabled/disabled
	ACK               bool                   // Ack packet
	FCnt              uint16                 // Frame counter downstream
	Port              uint8                  // Port for output. The port is set as the same time as the payload
	Payload           []byte                 // The (application) payload. Note: This does not include any MAC commands
	MACCommands       protocol.MACCommandSet // The MAC commands in the packet
	JoinAcceptPayload protocol.JoinAcceptPayload
}

func newFrameOutput(mtype protocol.MType) frameOutput {
	return frameOutput{
		MType:       mtype,
		MACCommands: protocol.NewMACCommandSet(mtype, 255),
		Payload:     make([]byte, 0),
	}
}

// FrameOutputBuffer buffers the payload (as bytes) and MAC commands that should
// be transmitted to the end-device in the next frame(s). If the payload or
// MAC commands doesn't fit into one frame it will be split into multiple parts.
type FrameOutputBuffer struct {
	frameData map[protocol.EUI]frameOutput // frameOutput structs keyed on DevAddr
	mutex     *sync.Mutex
}

// NewFrameOutputBuffer creates a new FrameOutputBuffer instance.
func NewFrameOutputBuffer() FrameOutputBuffer {
	return FrameOutputBuffer{
		frameData: make(map[protocol.EUI]frameOutput),
		mutex:     &sync.Mutex{},
	}
}

// AddMACCommand adds a new MAC command to the device aggregator. Returns error
// if there is no more room for MAC commands. The MAC Command will be sent as
// soon as possible.
// BUG(stalehd): Does not keep track of the mac command length
func (d *FrameOutputBuffer) AddMACCommand(deviceEUI protocol.EUI, cmd protocol.MACCommand) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	fd, exists := d.frameData[deviceEUI]
	if !exists {
		// Invariant: New device. Make it and add the command
		newData := newFrameOutput(protocol.UnconfirmedDataDown)
		if !newData.MACCommands.Add(cmd) {
			// This shouldn't fail but...
			logging.Warning("Couldn't add MAC command (0x%02x) to aggregator for device %s", cmd.ID(), deviceEUI)
		}
		d.frameData[deviceEUI] = newData
		return nil
	}

	// Invariant: Device exists, add to existing
	if !fd.MACCommands.Add(cmd) {
		logging.Warning("Couldn't add MAC command (0x%02x) to aggregator for device %s", cmd.ID(), deviceEUI)
	}
	d.frameData[deviceEUI] = fd
	return nil
}

// SetPayload sets (or overwrites) the existing payload. An error is returned
// if the payload can't be set.
func (d *FrameOutputBuffer) SetPayload(deviceEUI protocol.EUI, payload []byte, port uint8, ack bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	fd, exists := d.frameData[deviceEUI]
	if !exists {
		// Invariant: Device doesn't exist. Make it and set the payload
		fd = newFrameOutput(protocol.UnconfirmedDataDown)
	}

	// Invariant: Payload already exists. Replace (or set it)
	fd.Payload = payload
	fd.Port = port
	fd.MType = protocol.UnconfirmedDataDown
	if ack {
		fd.MType = protocol.ConfirmedDataDown
	}
	d.frameData[deviceEUI] = fd
}

// SetJoinAcceptPayload sets the JoinAccept payload that should be sent to the device.
func (d *FrameOutputBuffer) SetJoinAcceptPayload(deviceEUI protocol.EUI, payload protocol.JoinAcceptPayload) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	fd, exists := d.frameData[deviceEUI]
	if !exists {
		fd := newFrameOutput(protocol.JoinAccept)
		fd.JoinAcceptPayload = payload
		d.frameData[deviceEUI] = fd
	}

	fd.JoinAcceptPayload = payload
	fd.Port = 0
	fd.MType = protocol.JoinAccept
	d.frameData[deviceEUI] = fd
}

// GetPHYPayloadForDevice retrieves the next PHYPayload item for the device.
// If there's no data available for the device an error is returned. Note
// that this might not pull all of the data for the device.
// BUG(stalehd): Uses fixed max length for payload
func (d *FrameOutputBuffer) GetPHYPayloadForDevice(device *model.Device, context *FrameContext) (protocol.PHYPayload, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	fd, exists := d.frameData[device.DeviceEUI]
	if !exists {
		return protocol.PHYPayload{}, fmt.Errorf("there is no data to be sent for device %s", device.DeviceEUI)
	}

	payloadLength := len(fd.Payload)
	macLength := fd.MACCommands.Size()

	// Must have payload or MAC command unless this is a JoinAccept message or
	// there's a message ack pending
	if payloadLength == 0 && macLength == 0 && fd.MType != protocol.JoinAccept && !fd.ACK {
		// Remove the device from the map and return
		delete(d.frameData, device.DeviceEUI)
		return protocol.PHYPayload{}, errors.New("no data to send to device")
	}

	mhdr := protocol.MHDR{
		MType:        fd.MType,
		MajorVersion: protocol.MaxSupportedVersion,
	}

	fctrl := protocol.FCtrl{
		ADR:      fd.ADR,
		ACK:      fd.ACK,
		FPending: false,
		ClassB:   false,
		FOptsLen: 0,
	}
	// Set ack flag to false (since it will be sent)
	fd.ACK = false
	ret := protocol.NewPHYPayload(mhdr.MType)
	ret.MHDR = mhdr
	ret.MACPayload.FHDR.DevAddr = device.DevAddr
	ret.MACPayload.FHDR.FCtrl = fctrl
	ret.MACPayload.FHDR.FCnt = fd.FCnt
	ret.MACPayload.FPort = fd.Port
	ret.JoinAcceptPayload = fd.JoinAcceptPayload

	if payloadLength > 0 {
		// There's payload. Set it in the return value
		payloadSizes, err := context.GatewayContext.Radio.Band.MaximumPayload(context.GatewayContext.Radio.DataRate)
		if err != nil {
			return protocol.PHYPayload{}, err
		}

		var maxPayload uint8
		if macLength > 0 {
			maxPayload = payloadSizes.WithFOpts()
		} else {
			maxPayload = payloadSizes.WithoutFOpts()
		}
		if payloadLength > int(maxPayload) {
			ret.MACPayload.FRMPayload = fd.Payload[0:maxPayload]
			fd.Payload = fd.Payload[maxPayload:]

		} else {
			ret.MACPayload.FRMPayload = fd.Payload[:]
			fd.Payload = make([]byte, 0)
		}

		// Put MAC commands into the FOpts array
		list := fd.MACCommands.List()
		for len(list) > 0 && ret.MACPayload.FHDR.FOpts.Add(list[0]) {
			list = list[1:]
		}
	} else {
		// Put MAC commands into the payload
		ret.MACPayload.MACCommands.Copy(fd.MACCommands)
		fd.MACCommands.Clear()
	}

	if len(fd.Payload) > 0 || fd.MACCommands.Size() > 0 {
		ret.MACPayload.FHDR.FCtrl.FPending = true
	}

	if fd.MType == protocol.JoinAccept {
		// JoinAccept message is sent. There will be no more frames
		fd.MType = protocol.UnconfirmedDataDown
		fd.JoinAcceptPayload = protocol.JoinAcceptPayload{}
	}

	d.frameData[device.DeviceEUI] = fd
	return ret, nil
}

// SetMessageAckFlag sets the message acknowledgement flag. If the flag is set
// a message will be sent to the device at the earliest opportunity, regardless
// if there's payload or MAC frames to be sent.
func (d *FrameOutputBuffer) SetMessageAckFlag(deviceEUI protocol.EUI, ackFlag bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	fd, exists := d.frameData[deviceEUI]

	if !exists {
		// Invariant: Device doesn't exist. Make it and set the flag
		fd = newFrameOutput(protocol.UnconfirmedDataDown)
	}

	fd.ACK = ackFlag
	d.frameData[deviceEUI] = fd
}
