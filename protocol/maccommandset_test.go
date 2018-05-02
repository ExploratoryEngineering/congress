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
	"testing"
)

func TestMACList(t *testing.T) {
	list := NewMACCommandSet(ConfirmedDataUp, MaxFOptsLen)
	commandList := []CID{
		PingSlotChannelReq,
		BeaconFreqReq,
		LinkCheckAns,
		DevStatusAns,
		RXParamSetupAns,
		RXTimingSetupReq,
		LinkADRReq,
		PingSlotInfoReq,
	}

	if list.Size() != 0 {
		t.Fatalf("List isn't empty")
	}
	if list.Message() != ConfirmedDataUp {
		t.Fatal("List doesn't contain uplink values")
	}
	for i, v := range commandList {
		if !list.Add(NewUplinkMACCommand(v)) {
			t.Fatalf("Error adding command # %d", i)
		}
	}

	if list.Size() != len(commandList) {
		t.Fatalf("Length of list is incorrect. Expected %d but got %d", len(commandList), list.Size())
	}

	if !list.Contains(LinkCheckAns) {
		t.Fatal("Missing a command that was added")
	}

	list.Remove(LinkCheckAns)
	if list.Contains(LinkCheckAns) {
		t.Fatal("Command should no longer be in the list")
	}

	last := CID(0)
	for _, v := range list.List() {
		if v.ID() <= last {
			t.Fatal("Element is smaller")
		}
		last = v.ID()
	}

	// Store to a byte buffer
	buf := make([]byte, 1024)
	pos := 0
	if err := list.encode(buf, &pos); err != nil {
		t.Fatal("Got error encoding list")
	}

	list.Clear()

	pos = 0
	if err := list.decode(buf, &pos); err != nil && err != errUnknownMAC {
		t.Fatalf("got error decoding list: %v", err)
	}
	// All commands should be present
	list.Add(NewUplinkMACCommand(LinkCheckAns))
	for _, v := range commandList {
		if !list.Contains(v) {
			t.Fatalf("0x%02x is missing from the list", v)
		}
	}
}

func TestMACCommandListInvalidBuffer(t *testing.T) {
	commandSet := NewMACCommandSet(ConfirmedDataUp, MaxFOptsLen)
	commandList := []CID{
		PingSlotChannelReq,
		BeaconFreqReq,
		LinkCheckAns,
		DevStatusAns,
		RXParamSetupAns,
		RXTimingSetupReq,
		LinkADRReq,
		PingSlotInfoReq,
	}
	for _, v := range commandList {
		commandSet.Add(NewUplinkMACCommand(v))
	}
	basicDecoderTests(t, &commandSet)
	basicEncoderTests(t, &commandSet)
}

// Test decode output from a big set into a smaller set
func TestMACCommandSetTooBig(t *testing.T) {
	bigSet := NewMACCommandSet(ConfirmedDataUp, MaxFOptsLen)
	commandList := []CID{
		PingSlotFreqAns,
		BeaconFreqAns,
		LinkCheckAns,
		DevStatusAns,
		RXParamSetupAns,
		RXTimingSetupReq,
		LinkADRReq,
		PingSlotInfoReq,
	}
	for _, v := range commandList {
		if !bigSet.Add(NewUplinkMACCommand(v)) {
			t.Fatalf("Couldn't add %T (%+v) to big set", v, v)
		}
	}
	buf := make([]byte, 16)
	pos := 0
	bigSet.encode(buf, &pos)

	smallSet := NewMACCommandSet(ConfirmedDataUp, 4)
	pos = 0
	if err := smallSet.decode(buf[0:bigSet.EncodedLength()], &pos); err != nil {
		t.Fatal("Got error decoding into small set")
	}
	if smallSet.Size() == bigSet.Size() {
		t.Fatal("The small set shouldn't contain all of the big set")
	}

	// Encode the small set, decode into the big. That should yield an error
	pos = 0
	if err := smallSet.encode(buf, &pos); err != nil {
		t.Fatal("Couldn't encode the small set")
	}
	bigSet.Clear()
	pos = 0
	if err := bigSet.decode(buf[0:smallSet.EncodedLength()], &pos); err == nil {
		t.Fatal("Expected error when decoding small buffer into big set")
	}
}

// manually create a buffer with a series of MAC commands, add to set and attempt a decode
// they should do just fine
func TestMACComamndSetDecoding(t *testing.T) {
	mac1 := NewUplinkMACCommand(BeaconFreqReq) // CID = 0x13
	mac2 := NewUplinkMACCommand(LinkADRAns)    // CID = 0x03
	mac3 := NewUplinkMACCommand(DevStatusAns)  // CID = 0x06

	set := NewFOptsSet(ConfirmedDataUp)
	set.Add(mac1)
	set.Add(mac2)
	set.Add(mac3)

	// Encode in order. They should all appear a-ok
	buffer := make([]byte, 128)
	reset := func(buffer []byte) {
		for i := range buffer {
			buffer[i] = 0x88
		}
	}
	reset(buffer)
	pos := 0
	mac2.encode(buffer, &pos)
	mac3.encode(buffer, &pos)
	mac1.encode(buffer, &pos)

	pos = 0
	if err := set.decode(buffer, &pos); err != nil && err != errUnknownMAC {
		t.Fatalf("Couldn't decode set: %v", err)
	}

	if !set.Contains(BeaconFreqAns) || !set.Contains(LinkADRAns) || !set.Contains(DevStatusAns) {
		t.Fatal("Missing commands from result")
	}

	list := set.List()
	if list[0].ID() != LinkADRAns || list[1].ID() != DevStatusAns || list[2].ID() != BeaconFreqAns {
		t.Fatal("Order of items are screwed")
	}

	// Limit the count but keep buffer
	shortSet := NewMACCommandSet(ConfirmedDataUp, mac2.Length()+mac3.Length())
	pos = 0

	if err := shortSet.decode(buffer, &pos); err != nil {
		t.Fatal("Couldn't decode short set")
	}
	if shortSet.Size() != 2 {
		t.Fatalf("got incorrect number of commands. Would have %d but got %d", 2, shortSet.Size())
	}
	// Inject unknown CID into buffer, decode.
	buffer[mac2.Length()+mac3.Length()] = 0x44
	pos = 0
	if err := set.decode(buffer, &pos); err != errUnknownMAC {
		t.Fatal("Error decoding with unknown CID")
	}
	if set.Size() != 2 {
		t.Fatalf("Got more than 2 MAC commands (%d)", set.Size())
	}
	reset(buffer)
	pos = 0
	mac1.encode(buffer, &pos)

	pos = 0
	if err := set.decode(buffer, &pos); err != nil && err != errUnknownMAC {
		t.Fatal("Couldn't decode buffer with one element")
	}
	if set.Size() != 1 {
		t.Fatal("Set is too big")
	}
	if set.List()[0].ID() != BeaconFreqAns {
		t.Fatal("Couldn't find the command")
	}
}
