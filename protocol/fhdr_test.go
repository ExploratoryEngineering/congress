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

func TestFHDREncodeDecode(t *testing.T) {
	f1 := FHDR{
		DevAddr: DevAddr{NwkID: 1, NwkAddr: 2345},
		FCtrl:   FCtrl{},
		FCnt:    0xFFFF,
		FOpts:   NewFOptsSet(ConfirmedDataDown),
	}
	for _, v := range []MACCommand{
		NewDownlinkMACCommand(NewChannelReq),
		NewDownlinkMACCommand(DevStatusAns),
		NewDownlinkMACCommand(LinkADRAns),
		NewDownlinkMACCommand(LinkCheckReq)} {
		if !f1.FOpts.Add(v) {
			t.Fatalf("Couldn't add element %T (%+v) to fOpts", v, v)
		}
	}
	buffer := make([]byte, 1024)
	pos := 0
	if err := f1.encode(buffer, &pos); err != nil {
		t.Fatalf("Got error encoding FHDR: %v", err)
	}
	f2 := FHDR{FOpts: NewFOptsSet(ConfirmedDataDown)}
	pos = 0
	if err := f2.decode(buffer, &pos); err != nil {
		t.Fatalf("Got error decoding FHDR: %v", err)
	}

	compareMACCommands("f1/f2", &f1.FOpts, &f2.FOpts, t)
	if f1.DevAddr != f2.DevAddr || f1.FCnt != f2.FCnt || f1.FCtrl != f2.FCtrl {
		t.Fatalf("Encoded and decoded aren't equal: %+v != %+v", f1, f2)
	}
}

func TestFHDRBufferRangeChecks(t *testing.T) {
	basicEncoderTests(t, &FHDR{})
	basicDecoderTests(t, &FHDR{})
}

// Ensure truncated buffers doesn't break the decoder
func TestFHDRDecodeBufferLimit(t *testing.T) {
	f1 := FHDR{
		DevAddr: DevAddr{NwkID: 1, NwkAddr: 2345},
		FCtrl:   FCtrl{},
		FCnt:    0xFFFF,
		FOpts:   NewFOptsSet(ConfirmedDataDown),
	}
	for _, v := range []MACCommand{
		NewDownlinkMACCommand(DevStatusAns),
		NewDownlinkMACCommand(LinkCheckReq),
		NewDownlinkMACCommand(DutyCycleAns),
		NewDownlinkMACCommand(NewChannelAns),
		NewDownlinkMACCommand(RXTimingSetupAns),
		NewDownlinkMACCommand(PingSlotInfoReq)} {
		if !f1.FOpts.Add(v) {
			t.Fatalf("Got error adding %T %+v to command set", v, v)
		}
	}
	buffer := make([]byte, 1024)
	pos := 0
	if err := f1.encode(buffer, &pos); err != nil {
		t.Fatalf("Got error encoding FHDR: %v", err)
	}

	for i := 0; i < pos-1; i++ {
		f2 := FHDR{FOpts: NewFOptsSet(ConfirmedDataDown)}
		p := 0
		if err := f2.decode(buffer[0:i], &p); err == nil {
			t.Fatalf("Expected error with buffer that is %d bytes (of %d)", i, pos)
		}
	}
}
