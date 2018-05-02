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
	"github.com/ExploratoryEngineering/congress/gateway"
	"github.com/ExploratoryEngineering/congress/server"
)

// GwForwarder is the generic gateway interface. All of the communication is
// done via two channels. Outputs from the gateway is sent on the output channel
// while inputs are forwarded to the gateway.
type GwForwarder interface {
	// Start launches the forwarder and starts sending and receiving packets
	Start()
	// Stop terminates the forwarder and closes the input and output channels.
	Stop()
	// Input returns the input channel for the forwarder (ie data to send)
	Input() chan<- server.GatewayPacket
	// Output returns the output channel for the forwarder (ie data received)
	Output() <-chan server.GatewayPacket
}

// NewGwForwarder creates a new gateway forwarder instance. The input and
// output channels
func NewGwForwarder(port int, context *server.Context) GwForwarder {
	return gateway.NewGenericPacketForwarder(port, context.Storage.Gateway, context)
}
