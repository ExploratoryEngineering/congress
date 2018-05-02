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
	"errors"
	"flag"
	"fmt"
)

// Params is a struct with the command line parameters, in effect
// the configuration for Eagle One
type Params struct {
	DeviceCount        int
	DeviceMessages     int
	CorruptMIC         int
	CorruptedPayload   int
	DuplicateMessages  int
	TransmissionDelay  int
	UDPPort            int
	MaxPayloadSize     int
	NumericalPayload   bool
	AppEUI             string
	GatewayEUI         string
	ListSent           bool
	FrameCounterErrors int
	KeepApplication    bool
	KeepDevices        bool
	KeepGateway        bool
	Mode               string
	Hostname           string
	LogLevel           int
	NetID              uint32 // note: no parameter for this. Using the default (0)
	MQTTLocalEndpoint  string
	MQTTLocalPort      int
	MQTTEndpoint       string // Integration test config
	MQTTPort           int
	MQTTUsername       string
	MQTTPassword       string
	MQTTTLS            bool
}

// Valid validate the parameters
func (p *Params) Valid() error {
	if p.CorruptMIC > 100 {
		return errors.New("MIC must be in the range 0-100")
	}
	if p.CorruptedPayload > 100 {
		return errors.New("CorruptedPayload must be in the range 0-100")
	}
	if p.DuplicateMessages > 100 {
		return errors.New("DuplicateMessages must be in the range 0-100")
	}
	if p.FrameCounterErrors > 100 {
		return errors.New("Frame counter errors should be in the range 0-100")
	}

	if p.DeviceCount < 0 {
		return fmt.Errorf("# of devices must be >= 0")
	}
	if p.DeviceMessages < 0 {
		return fmt.Errorf("# device messages must be >= 0")
	}
	if p.LogLevel < 0 || p.LogLevel > 3 {
		return fmt.Errorf("Unknown log level valid values are (from low to high) 0, 1, 2 or 3")
	}
	return nil
}

// CommandLineParameters is the parameters supplied via the command line
var CommandLineParameters Params

func init() {
	flag.IntVar(&CommandLineParameters.DeviceCount, "devices", 10, "Number of devices.")
	flag.IntVar(&CommandLineParameters.DeviceMessages, "messages", 10, "Number of messages from each device.")
	flag.IntVar(&CommandLineParameters.CorruptMIC, "corrupt-mic", 5, "Percentage of packets generated that has a corrupt checksum.")
	flag.IntVar(&CommandLineParameters.CorruptedPayload, "corrupt-payload", 0, "Percentage of packets generated that has a corrupt checksum.")
	flag.IntVar(&CommandLineParameters.DuplicateMessages, "duplicate-message", 2, "Percentage of messages that will be duplicated.")
	flag.IntVar(&CommandLineParameters.TransmissionDelay, "transmission-delay", 1000, "Delay (in milliseconds) between transmissions.")
	flag.IntVar(&CommandLineParameters.UDPPort, "congress-udp-port", 8000, "Congress port")
	flag.IntVar(&CommandLineParameters.MaxPayloadSize, "max-payload-size", 222, "ID offset for device EUI")
	flag.BoolVar(&CommandLineParameters.NumericalPayload, "fancy-numerical-payload", false, "Generates non-insane numerical output in the form of a two bytes")
	flag.StringVar(&CommandLineParameters.AppEUI, "application-eui", "", "Use existing application (-keep-application will be ignored)")
	flag.StringVar(&CommandLineParameters.GatewayEUI, "gateway-eui", "", "Use existing gateway (--keep-gateway will be ignored)")
	flag.IntVar(&CommandLineParameters.FrameCounterErrors, "frame-counter-errors", 5, "Frame counter errors (0-100)")
	flag.BoolVar(&CommandLineParameters.KeepApplication, "keep-application", false, "Keep application when shutting down, don't remove it")
	flag.BoolVar(&CommandLineParameters.KeepGateway, "keep-gateway", false, "Keep gateway when shutting down, don't delete it.")
	flag.BoolVar(&CommandLineParameters.KeepDevices, "keep-devices", false, "Keep devices when shutting down")
	flag.StringVar(&CommandLineParameters.Mode, "mode", "batch", "Eagle One mode (interactive, batch, test)")
	flag.IntVar(&CommandLineParameters.LogLevel, "loglevel", 1, "Log level (0: Debug, 1: Info, 2: Warning: 3: Error)")
	flag.StringVar(&CommandLineParameters.MQTTEndpoint, "test-mqtt-endpoint", "mqtt", "MQTT broker endpoint")
	flag.IntVar(&CommandLineParameters.MQTTPort, "test-mqtt-port", 1883, "MQTT broker port")
	flag.StringVar(&CommandLineParameters.MQTTLocalEndpoint, "test-mqtt-local-endpoint", "localhost", "MQTT broker endpoint")
	flag.IntVar(&CommandLineParameters.MQTTLocalPort, "test-mqtt-local-port", 1883, "MQTT broker port")
	flag.StringVar(&CommandLineParameters.MQTTUsername, "test-mqtt-username", "test1", "MQTT broker username")
	flag.StringVar(&CommandLineParameters.MQTTPassword, "test-mqtt-password", "test1", "MQTT broker password")
	flag.BoolVar(&CommandLineParameters.MQTTTLS, "test-mqtt-tls", false, "MQTT broker TLS flag")
	flag.Parse()
}
