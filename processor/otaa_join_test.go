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

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage/memstore"
)

func TestOTAAJoinRequestProcessing(t *testing.T) {

	deviceEUI, _ := protocol.EUIFromString("00-01-02-03-04-05-06-07")
	appEUI, _ := protocol.EUIFromString("00-01-02-03-04-05-06-08")

	store := memstore.CreateMemoryStorage(0, 0)
	application := model.Application{
		AppEUI: appEUI,
		Tags:   model.NewTags(),
	}
	device := model.Device{
		DeviceEUI:       deviceEUI,
		AppEUI:          appEUI,
		State:           model.OverTheAirDevice,
		FCntUp:          100,
		FCntDn:          100,
		RelaxedCounter:  false,
		DevNonceHistory: make([]uint16, 0),
	}

	store.Application.Put(application, model.SystemUserID)
	store.Device.Put(device, appEUI)

	inputChan := make(chan server.LoRaMessage)

	foBuffer := server.NewFrameOutputBuffer()
	decrypter := NewDecrypter(&server.Context{
		Storage:     &store,
		FrameOutput: &foBuffer,
		Config:      &server.Configuration{},
	}, inputChan)

	payload := protocol.NewPHYPayload(protocol.JoinRequest)
	payload.JoinRequestPayload = protocol.JoinRequestPayload{
		DevEUI:   deviceEUI,
		AppEUI:   appEUI,
		DevNonce: 0x0102,
	}
	input := server.LoRaMessage{
		Payload: payload,
		FrameContext: server.FrameContext{
			Device:         model.NewDevice(),
			Application:    model.NewApplication(),
			GatewayContext: server.GatewayPacket{},
		},
	}

	close(inputChan)

	// This should result in an output message
	go decrypter.processJoinRequest(input)

	select {
	case <-decrypter.Output():
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Did not get output on output channel!")
	}
}
