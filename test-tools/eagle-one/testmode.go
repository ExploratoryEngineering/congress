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
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/logging"
	"github.com/telenordigital/lassie-go"

	"github.com/eclipse/paho.mqtt.golang"
)

// TestMode is an integration test mode for E1.
type TestMode struct {
	Config      Params
	Gateway     lassie.Gateway
	Application lassie.Application
	APIDevices  []lassie.Device
	Devices     []*EmulatedDevice
	Client      *lassie.Client
	Errors      []string
}

const numTestDevices = 10

// Prepare creates the necessary resources in Congress
func (t *TestMode) Prepare(congress *lassie.Client, app lassie.Application, gw lassie.Gateway) error {
	t.Errors = make([]string, 0)
	t.Application = app
	t.Gateway = gw
	logging.Info("Creating devices")
	t.APIDevices = make([]lassie.Device, 0)
	// Make devices
	for i := 0; i < numTestDevices; i++ {
		template := lassie.Device{Type: "OTAA"}
		if rand.Intn(2) == 0 {
			template.Type = "ABP"
		}
		device, err := congress.CreateDevice(app.EUI, template)
		if err != nil {
			t.Errors = append(t.Errors, err.Error())
			return err
		}
		t.APIDevices = append(t.APIDevices, device)
	}
	t.Client = congress
	return nil
}

// Cleanup removes the created resources
func (t *TestMode) Cleanup(congress *lassie.Client, app lassie.Application, gw lassie.Gateway) {
	logging.Info("Removing devices")
	for _, d := range t.APIDevices {
		if err := congress.DeleteDevice(t.Application.EUI, d.EUI); err != nil {
			t.Errors = append(t.Errors, err.Error())
			logging.Warning("Unable to delete device with EUI %s", d.EUI)
			// TODO: Register as error
		}
	}
}

func (t *TestMode) reportError(err error) {
	logging.Error("Error: %v", err)
	t.Errors = append(t.Errors, err.Error())
}

// Create emulated devices and join if required
func (t *TestMode) createDevices(gatewayChannel chan string, publisher *EventRouter) {
	// Create emulated devices
	t.Devices = make([]*EmulatedDevice, 0)
	for _, device := range t.APIDevices {
		keys, err := NewDeviceKeys(t.Application.EUI, device)
		if err != nil {
			logging.Warning("couldn't create device keys: %v", err)
			continue
		}
		newDevice := NewEmulatedDevice(t.Config, keys, gatewayChannel, publisher)
		if device.Type == "OTAA" {
			if err := newDevice.Join(joinAttempts); err != nil {
				t.reportError(fmt.Errorf("Device %s couldn't join: %v", device.EUI, err))
				continue
			}
		}
		t.Devices = append(t.Devices, newDevice)
	}
}

// Schedule downstream messages for devices. These will be sent later.
func (t *TestMode) sendDownstreamMessages(payload string) (sentMessages int32) {
	for index, device := range t.APIDevices {
		msg := lassie.DownstreamMessage{
			Port:       uint8(index + 1),
			Ack:        false,
			HexPayload: payload,
		}
		if err := t.Client.ScheduleMessage(t.Application.EUI, device.EUI, msg); err != nil {
			t.reportError(fmt.Errorf("Couldn't schedule message for device %s: %v", device.EUI, err))
		} else {
			sentMessages++
			logging.Info("Scheduled downstream message for device %s", device.EUI)
		}
	}
	return
}

// Send upstream messages and receive downstream messages sent earlier
func (t *TestMode) sendUpstreamMessages(expectedPayload string) (sent int32, received int32) {
	for i, e := range t.Devices {
		payload := []byte{byte(i)}
		if err := e.SendMessageWithPayload(protocol.UnconfirmedDataUp, payload); err != nil {
			t.reportError(fmt.Errorf("Device %s couldn't send message: %v", e.keys.DevEUI, err))
			continue
		}
		if len(e.ReceivedMessages) == 0 {
			t.reportError(fmt.Errorf("Device %s did not receive a scheduled message", e.keys.DevEUI))
		} else {
			if strings.ToUpper(e.ReceivedMessages[0].Payload) != expectedPayload {
				t.reportError(fmt.Errorf("Not the payload I expected. Expected %s but got %s", expectedPayload, e.ReceivedMessages[0].Payload))
			}
			received++
		}
		sent++
	}
	return
}

// Run launches the test.
func (t *TestMode) Run(gatewayChannel chan string, publisher *EventRouter, app lassie.Application, gw lassie.Gateway) {
	websocketReceived := int32(0)

	// Start listening on the application stream websocket. All messages sent
	// by the devices should be forwarded on this.
	go func() {
		if err := t.Client.ApplicationStream(t.Application.EUI, func(data lassie.DeviceData) {
			logging.Info("Websocket: DevAddr %s sent %v", data.DeviceAddress, data.HexData)
			atomic.AddInt32(&websocketReceived, 1)
		}); err != nil {
			t.reportError(fmt.Errorf("Data stream error: %v", err))
		}
	}()

	outputConfig := lassie.MQTTConfig{
		Endpoint:  t.Config.MQTTEndpoint,
		Port:      t.Config.MQTTPort,
		Username:  t.Config.MQTTUsername,
		Password:  t.Config.MQTTPassword,
		TLS:       t.Config.MQTTTLS,
		TopicName: "EagleOne",
		ClientID:  "EagleOne",
	}
	_, err := t.Client.CreateOutput(t.Application.EUI, &outputConfig)
	if err != nil {
		t.reportError(fmt.Errorf("Unable to create output: %v", err))
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	mqttMessageCount := int32(0)
	// Start listening on the topic.
	go func(wg *sync.WaitGroup) {
		options := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", t.Config.MQTTLocalEndpoint, t.Config.MQTTLocalPort))
		options.SetClientID("e1client")
		options.SetUsername(t.Config.MQTTUsername).SetPassword(t.Config.MQTTPassword)

		client := mqtt.NewClient(options)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			t.reportError(token.Error())
			return
		}
		logging.Info("Connected to MQTT broker. Subscribing to messages")
		client.Subscribe("EagleOne", 0, func(client mqtt.Client, msg mqtt.Message) {
			var dataMsg lassie.DeviceData
			if err := json.Unmarshal(msg.Payload(), &dataMsg); err != nil {
				t.reportError(err)
				return
			}
			atomic.AddInt32(&mqttMessageCount, 1)
			// Check payload here.
			logging.Info("MQTT:Received message from device %s with payload %s", dataMsg.DeviceEUI, dataMsg.HexData)
		})
		wg.Wait()
		defer client.Disconnect(0)
	}(wg)

	// Get topic name
	t.createDevices(gatewayChannel, publisher)

	const DownstreamPayload = "0123456789ABCDEF"
	sentMessages := t.sendDownstreamMessages(DownstreamPayload)
	sent, received := t.sendUpstreamMessages(DownstreamPayload)

	wg.Done()
	receivedFromWS := atomic.LoadInt32(&websocketReceived)
	if receivedFromWS != sent {
		t.reportError(fmt.Errorf("Sent %d messages but got only %d on websocket", sent, received))
	}
	if received != sentMessages {
		t.reportError(fmt.Errorf("Scheduled %d messages but the devices only received %d in total", sentMessages, received))
	}
	if mqttMessageCount != sent {
		t.reportError(fmt.Errorf("Sent %d message but only %d was received via MQTT", sent, mqttMessageCount))
	}
}

// Failed returns true if the mode has failed
func (t *TestMode) Failed() bool {
	return len(t.Errors) != 0
}
