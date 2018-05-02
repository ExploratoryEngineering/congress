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
	"math/rand"
	"testing"

	"bytes"

	"github.com/ExploratoryEngineering/congress/band"
	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
)

//
// This is the common context used by all of the tests. The aggregator uses
// the band to determine the maximum payload size
var context = FrameContext{
	GatewayContext: GatewayPacket{
		Radio: RadioContext{
			Band:      band.EU868{},
			DataRate:  "SF10BW125",
			Frequency: 868.1,
		},
	},
}

func makeRandomDevAddr() protocol.DevAddr {
	return protocol.DevAddr{
		NwkID:   uint8(rand.Int() % 0x1F),
		NwkAddr: uint32(rand.Int() % 0xFFFFFF),
	}
}

func getRandomPort() uint8 {
	return uint8(rand.Int() % 0xFF)
}

func TestDeviceAggregatorPayload(t *testing.T) {
	agg := NewFrameOutputBuffer()
	d1 := &model.Device{DeviceEUI: makeRandomEUI(), DevAddr: makeRandomDevAddr()}
	d2 := &model.Device{DeviceEUI: makeRandomEUI(), DevAddr: makeRandomDevAddr()}

	setPayload := func(d protocol.EUI, p []byte) {
		agg.SetPayload(d, p, getRandomPort(), false)
	}

	comparePayload := func(d *model.Device, p []byte, id string) {
		payload, err := agg.GetPHYPayloadForDevice(d, &context)
		if err != nil {
			t.Fatalf("Got error retrieving payload %s: %v", id, err)
		}
		if !bytes.Equal(payload.MACPayload.FRMPayload, p) {
			t.Fatalf("Mismatched payload %s: len(1): %v, len(2):%v", id, len(payload.MACPayload.FRMPayload), len(p))
		}

	}

	p1 := make([]byte, 16)
	rand.Read(p1)
	p2 := make([]byte, 32)
	rand.Read(p2)

	setPayload(d1.DeviceEUI, p1)
	setPayload(d2.DeviceEUI, p2)

	comparePayload(d1, p1, "p1/1")
	comparePayload(d2, p2, "p2/1")

	maxPayload, err := context.GatewayContext.Radio.Band.MaximumPayload(context.GatewayContext.Radio.DataRate)
	if err != nil {
		t.Fatal("Error getting max payload: ", err)
	}
	p1 = make([]byte, maxPayload.WithoutFOpts())
	rand.Read(p1)
	p2 = make([]byte, maxPayload.WithoutFOpts())
	rand.Read(p2)

	setPayload(d1.DeviceEUI, p2)
	setPayload(d1.DeviceEUI, p1)
	setPayload(d2.DeviceEUI, p1)
	setPayload(d2.DeviceEUI, p2)

	comparePayload(d1, p1, "p1/2")
	comparePayload(d2, p2, "p2/2")

	d3 := &model.Device{DeviceEUI: makeRandomEUI(), DevAddr: makeRandomDevAddr()}
	if _, err := agg.GetPHYPayloadForDevice(d3, &context); err == nil {
		t.Fatal("Expected error when retrieving device that doesn't exist")
	}
}

func TestDeviceAggregatorMAC(t *testing.T) {
	agg := NewFrameOutputBuffer()
	d1 := &model.Device{DeviceEUI: makeRandomEUI(), DevAddr: makeRandomDevAddr()}
	d2 := &model.Device{DeviceEUI: makeRandomEUI(), DevAddr: makeRandomDevAddr()}

	if err := agg.AddMACCommand(d1.DeviceEUI, protocol.NewDownlinkMACCommand(protocol.LinkCheckReq)); err != nil {
		t.Fatal("Error adding MAC command 1: ", err)
	}

	if err := agg.AddMACCommand(d1.DeviceEUI, protocol.NewDownlinkMACCommand(protocol.LinkADRAns)); err != nil {
		t.Fatal("Error adding MAC command 2: ", err)
	}

	if err := agg.AddMACCommand(d1.DeviceEUI, protocol.NewDownlinkMACCommand(protocol.PingSlotInfoReq)); err != nil {
		t.Fatal("Error adding MAC command 3: ", err)
	}

	if err := agg.AddMACCommand(d2.DeviceEUI, protocol.NewDownlinkMACCommand(protocol.DevStatusAns)); err != nil {
		t.Fatal("Error adding MAC command 4: ", err)
	}

	if err := agg.AddMACCommand(d2.DeviceEUI, protocol.NewDownlinkMACCommand(protocol.NewChannelAns)); err != nil {
		t.Fatal("Error adding MAC command 5: ", err)
	}

	do1, err := agg.GetPHYPayloadForDevice(d1, &context)
	if err != nil {
		t.Fatal("Couldn't retrieve output for d1: ", err)
	}
	if do1.MACPayload.MACCommands.Size() != 3 {
		t.Fatalf("not the expected number of MAC commands for d1: %v (expected 3)", do1.MACPayload.MACCommands.Size())
	}

	if !do1.MACPayload.MACCommands.Contains(protocol.LinkCheckReq) &&
		!do1.MACPayload.MACCommands.Contains(protocol.LinkADRAns) &&
		!do1.MACPayload.MACCommands.Contains(protocol.PingSlotInfoReq) {
		t.Fatal("Didn't find LinkCheckReq MAC command")
	}

	do2, err := agg.GetPHYPayloadForDevice(d2, &context)
	if err != nil {
		t.Fatal("Couln't retrieve output for d2: ", err)
	}

	if do2.MACPayload.MACCommands.Size() != 2 {
		t.Fatalf("not the expected number of MAC commands for d2: %v (expected 2)", do2.MACPayload.MACCommands.Size())
	}

	if !do2.MACPayload.MACCommands.Contains(protocol.DevStatusAns) &&
		!do2.MACPayload.MACCommands.Contains(protocol.NewChannelAns) {
		t.Fatal("Didn't find any DutyCycleReq commands in the d2 output")
	}

}

// Make sure you can't read unknown devices
func TestNonexistingMessage(t *testing.T) {
	da := NewFrameOutputBuffer()
	d := &model.Device{DeviceEUI: makeRandomEUI(), DevAddr: makeRandomDevAddr()}
	if _, err := da.GetPHYPayloadForDevice(d, &context); err == nil {
		t.Fatal("Did not expect to get PHYPayload for unknown device")
	}
}

// Ensure a big message is split into properly sized parts
func TestBigMessage(t *testing.T) {
	da := NewFrameOutputBuffer()

	d := &model.Device{DeviceEUI: makeRandomEUI(), DevAddr: makeRandomDevAddr()}
	buffer := make([]byte, 255)
	if n, err := rand.Read(buffer); n != len(buffer) || err != nil {
		t.Fatal("Couldn't generate random numbers")
	}

	da.SetPayload(d.DeviceEUI, buffer, 12, false)
	da.AddMACCommand(d.DeviceEUI, &protocol.MACLinkADRReq{})

	// Retrieve the messages until there's no more
	var returnedMacs []protocol.MACCommand
	var returnedBuffer []byte

	var err error
	var ret protocol.PHYPayload
	iterations := 0
	for err == nil {
		ret, err = da.GetPHYPayloadForDevice(d, &context)
		if err != nil {
			break
		}
		if iterations > 999 {
			t.Errorf("Breaking after %d iterations", iterations)
			break
		}
		iterations++
		returnedMacs = append(returnedMacs, ret.MACPayload.MACCommands.List()...)
		returnedMacs = append(returnedMacs, ret.MACPayload.FHDR.FOpts.List()...)
		returnedBuffer = append(returnedBuffer, ret.MACPayload.FRMPayload...)
	}
	if iterations == 0 {
		t.Fatalf("Didn't get anything at all, err = %s", err)
	}
	if !bytes.Equal(buffer, returnedBuffer) {
		t.Fatalf("Mismatch on returned buffer (orig len=%d, returned len=%d)", len(buffer), len(returnedBuffer))
	}
}

// Test message setting. The message type can be set multiple times and will
// issue a warning in the logs if the message type is overwritten.
func TestSetMessageType(t *testing.T) {
	da := NewFrameOutputBuffer()

	d := &model.Device{DeviceEUI: makeRandomEUI(), DevAddr: makeRandomDevAddr()}

	da.SetPayload(d.DeviceEUI, []byte{0, 1, 2, 3}, 1, false)

	ret, err := da.GetPHYPayloadForDevice(d, &context)
	if err != nil {
		t.Fatal("Got error retrieving phy payload: ", err)
	}

	if ret.MHDR.MType != protocol.UnconfirmedDataDown {
		t.Fatalf("Didn't get the expected message type. Expected %d got %d", protocol.UnconfirmedDataDown, ret.MHDR.MType)
	}

	if ret.MACPayload.FPort != 1 {
		t.Fatalf("Didn't get the expected port. Expected 1 got %d", ret.MACPayload.FPort)
	}

	da.SetPayload(d.DeviceEUI, []byte{0, 1, 2, 3, 4}, 2, false)

	da.SetPayload(d.DeviceEUI, []byte{0, 1, 2, 3, 4}, 3, true)

	ret, err = da.GetPHYPayloadForDevice(d, &context)
	if err != nil {
		t.Fatal("Got error retrieving phy payload: ", err)
	}
	if ret.MHDR.MType != protocol.ConfirmedDataDown {
		t.Fatalf("Didn't get the expected message type. Expected %d got %d", protocol.ConfirmedDataDown, ret.MHDR.MType)
	}

	if ret.MACPayload.FPort != 3 {
		t.Fatalf("Didn't get the expected port. Expected 3 got %d", ret.MACPayload.FPort)
	}

}
