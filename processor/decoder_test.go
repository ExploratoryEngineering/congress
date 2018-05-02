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
	"encoding/base64"
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/server"
)

// Ensure the decoder behaves when one channel closes
func TestDecoderChannels(t *testing.T) {
	context := server.Context{}
	input := make(chan server.GatewayPacket)
	decoder := NewDecoder(&context, input)

	go decoder.Start()

	close(input)
	select {
	case _, ok := <-decoder.Output():
		if ok {
			t.Fatal("Shouldn't be able to read from the channel")
		}

	}
}

func TestDecoderProcessing(t *testing.T) {
	s := NewStorageTestContext()
	context := server.Context{Storage: &s}
	input := make(chan server.GatewayPacket)
	decoder := NewDecoder(&context, input)

	go decoder.Start()

	// Send one message of bytes
	bytes, _ := base64.StdEncoding.DecodeString(GatewayValidPacket)
	input <- server.GatewayPacket{
		RawMessage: bytes,
	}

	// Ensure this message is received
	select {
	case <-decoder.Output():
		// This is OK.
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for decoded packet")
	}

	bytes, _ = base64.StdEncoding.DecodeString(GatewayInvalidPacket)
	input <- server.GatewayPacket{
		RawMessage: bytes,
	}
	select {
	case tmp := <-decoder.Output():
		t.Fatal("I did not expect to get that packet: ", tmp)
	case <-time.After(100 * time.Millisecond):
		// This is OK.
	}
	// Shut it down
	close(input)
}
