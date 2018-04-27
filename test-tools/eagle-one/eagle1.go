package main

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
	"fmt"
	"math/rand"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/logging"
	lassie "github.com/telenordigital/lassie-go"
)

// Eagle1 is the main testing tool. It will manage all of the infrastructure
// with the application,  gateway and packet forwarding. Message routing is done
// through the event router. It will publish events based on the device address.
type Eagle1 struct {
	Congress       *lassie.Client
	Config         Params
	Application    lassie.Application
	Gateway        lassie.Gateway
	Publisher      *EventRouter
	GatewayChannel chan string
	forwarder      *SyntheticForwarder
	shutdown       chan bool
}

func (e *Eagle1) newRandomEUI() string {
	octets := make([]byte, 8)
	rand.Read(octets)
	return fmt.Sprintf("%02x-%02x-%02x-%02x-%02x-%02x-%02x-%02x",
		octets[0], octets[1], octets[2], octets[3], octets[4], octets[5], octets[6], octets[7])
}

// Setup runs the setup procedures
func (e *Eagle1) Setup() error {
	var err error
	if e.Config.AppEUI == "" {
		newApp := lassie.Application{
			Tags: make(map[string]string),
		}
		newApp.Tags["name"] = "Eagle One Test Application"
		if e.Application, err = e.Congress.CreateApplication(newApp); err != nil {
			return fmt.Errorf("unable to create application in Congress: %v", err)
		}
	} else {
		e.Config.KeepApplication = true
		if e.Application, err = e.Congress.Application(e.Config.AppEUI); err != nil {
			return fmt.Errorf("Couldn't read application %s: %v", e.Config.AppEUI, err)
		}
	}

	if e.Config.GatewayEUI == "" {
		newGw := lassie.Gateway{
			EUI:       e.newRandomEUI(),
			IP:        "127.0.0.1",
			StrictIP:  false,
			Latitude:  50.3672,
			Longitude: 6.932,
			Altitude:  476.0,
		}
		if e.Gateway, err = e.Congress.CreateGateway(newGw); err != nil {
			return fmt.Errorf("Unable to create gateway in Congress: %v", err)
		}
	} else {
		e.Config.KeepGateway = true
		if e.Gateway, err = e.Congress.Gateway(e.Config.GatewayEUI); err != nil {
			return fmt.Errorf("Cannot retrieve a gateway with the EUI %s: %v", e.Config.GatewayEUI, err)
		}
	}
	logging.Info("Gateway EUI: %s", e.Gateway.EUI)
	logging.Info("Application EUI: %s", e.Application.EUI)

	return nil
}

// Teardown does a controlled terardown and removes application and gateway if needed.
func (e *Eagle1) Teardown() {
	e.shutdown <- true
	if !e.Config.KeepApplication {
		logging.Info("Removing application %s", e.Application.EUI)
		e.Congress.DeleteApplication(e.Application.EUI)
	}
	if !e.Config.KeepGateway {
		logging.Info("Removing gateway %s", e.Gateway.EUI)
		e.Congress.DeleteGateway(e.Gateway.EUI)
	}

}

// Run runs through the mode (batch/interactive)
func (e *Eagle1) Run(mode E1Mode) error {
	defer mode.Cleanup(e.Congress, e.Application, e.Gateway)
	if err := mode.Prepare(e.Congress, e.Application, e.Gateway); err != nil {
		return err
	}
	mode.Run(e.GatewayChannel, e.Publisher, e.Application, e.Gateway)
	return nil
}

func (e *Eagle1) decodingLoop() {
	for msg := range e.forwarder.OutputChannel() {
		p := protocol.NewPHYPayload(protocol.Proprietary)
		if err := p.UnmarshalBinary(msg); err != nil {
			logging.Warning("Unable to unmarshal message from gateway: %v", err)
			continue
		}
		e.Publisher.Publish(p.MACPayload.FHDR.DevAddr, p, msg)
	}
}

// StartForwarder launches a synthetic packet forwarder
func (e *Eagle1) StartForwarder() {
	e.shutdown = make(chan bool)
	e.forwarder = NewSyntheticForwarder(
		e.GatewayChannel, e.shutdown,
		e.Gateway.EUI, e.Config.Hostname,
		e.Config.UDPPort)

	logging.Info("Launching synthetic forwarder")
	go e.forwarder.Start()
	go e.decodingLoop()
}
