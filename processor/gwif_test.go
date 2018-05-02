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

	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/utils"
)

// Test strings, base 64 encoded.
const (
	// GatewayValidPacket contains a base64 encoded LoRa message
	GatewayValidPacket string = "gOZy5gGAAQALqBJvwTWKKB0="
	// GatewayInvalidPacket contains the string "foo bar baz" base64 encoded
	GatewayInvalidPacket string = "Zm9vIGJhciBiYXo="
)

// Test proper behaviour wrt channels. When the input channel is closed
// it should close the output channel.
func TestInterfaceChannels(t *testing.T) {
	s := NewStorageTestContext()
	context := server.Context{Storage: &s}

	port, err := utils.FreePort()
	if err != nil {
		t.Fatal("Could not get a random port: ", err)
	}
	gwif := NewGwForwarder(port, &context)

	go gwif.Start()

	if gwif.Input() == nil {
		t.Fatal("Should have an input channel")
	}
	if gwif.Output() == nil {
		t.Fatal("Should have an output channel")
	}

	gwif.Stop()
}
