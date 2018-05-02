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
)

func TestMacprocessorChannels(t *testing.T) {
	context := server.Context{}
	input := make(chan server.LoRaMessage)

	macprocessor := NewMACProcessor(&context, input)

	go macprocessor.Start()

	close(input)
	select {
	case _, ok := <-macprocessor.CommandNotifier():
		if ok {
			t.Fatal("Expected output channel to be closed")
		}
	}
}

func makeLoRaMessage(uplink bool, mType protocol.MType, fopts []protocol.MACCommand, payload []protocol.MACCommand) server.LoRaMessage {
	ret := server.LoRaMessage{Payload: protocol.NewPHYPayload(mType)}
	for _, v := range fopts {
		ret.Payload.MACPayload.FHDR.FOpts.Add(v)
	}
	for _, v := range payload {
		ret.Payload.MACPayload.MACCommands.Add(v)

	}
	return ret
}

// The MAC processor should forward the notification to the scheduler when
// a new LoRaMessage arrives. Ensure it does that while throwing all different
// sorts of MAC commands at it.
func TestMacprocessorForwarding(t *testing.T) {
	context := server.Context{}
	input := make(chan server.LoRaMessage)

	defer close(input)

	macprocessor := NewMACProcessor(&context, input)

	go macprocessor.Start()

	input <- makeLoRaMessage(true, protocol.UnconfirmedDataUp,
		[]protocol.MACCommand{
			&protocol.MACNewChannelAns{},
			&protocol.MACDevStatusAns{},
			&protocol.MACLinkCheckReq{},
			&protocol.MACDutyCycleAns{},
			&protocol.MACRXParamSetupAns{},
			&protocol.MACRXTimingSetupAns{},
		},
		[]protocol.MACCommand{
			&protocol.MACPingSlotFreqAns{},
			&protocol.MACLinkADRAns{},
			&protocol.MACPingSlotInfoReq{},
			&protocol.MACBeaconTimingReq{},
			&protocol.MACBeaconFreqAns{},
		})

	select {
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Did not get a notification for new command after 100ms")
	case <-macprocessor.CommandNotifier():
		// OK - got message
	}
}
