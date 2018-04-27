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
	"sync"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/telenordigital/lassie-go"
)

// BatchMode processing. Create
type BatchMode struct {
	Config           Params
	Application      lassie.Application
	devices          []lassie.Device
	OutgoingMessages chan string
	Publisher        *EventRouter
}

// Prepare prepares the processing
func (b *BatchMode) Prepare(congress *lassie.Client, app lassie.Application, gw lassie.Gateway) error {
	b.devices = make([]lassie.Device, 0)

	randomizer := NewRandomizer(50)

	for i := 0; i < int(b.Config.DeviceCount); i++ {
		t := "OTAA"
		randomizer.Maybe(func() {
			t = "ABP"
		})
		newDevice := lassie.Device{
			Type: t,
		}
		dev, err := congress.CreateDevice(app.EUI, newDevice)
		if err != nil {
			return fmt.Errorf("Unable to create device in Congress: %v", err)
		}
		b.devices = append(b.devices, dev)
	}
	logging.Info("# devices: %d", b.Config.DeviceCount)
	logging.Info("# messages: %d (total: %d)", b.Config.DeviceMessages, b.Config.DeviceCount*b.Config.DeviceMessages)
	return nil
}

// Cleanup resources after use. Remove devices if required.
func (b *BatchMode) Cleanup(congress *lassie.Client, app lassie.Application, gw lassie.Gateway) {
	if !b.Config.KeepDevices {
		logging.Info("Removing %d devices", len(b.devices))
		for _, d := range b.devices {
			congress.DeleteDevice(app.EUI, d.EUI)
		}
	}
}

// The number of join attempts before giving up
const joinAttempts = 5

func (b *BatchMode) launchDevice(device lassie.Device, wg *sync.WaitGroup) {
	keys, err := NewDeviceKeys(b.Application.EUI, device)
	if err != nil {
		logging.Warning("Got error converting lassie data into proper types: ", err)
	}

	generator := NewMessageGenerator(b.Config)
	remoteDevice := NewEmulatedDevice(
		b.Config,
		keys,
		b.OutgoingMessages,
		b.Publisher)

	defer wg.Done()
	// Join if needed
	if device.Type == "OTAA" {
		if err := remoteDevice.Join(joinAttempts); err != nil {
			logging.Warning("Device %s couldn't join after %d attempts", device.EUI, joinAttempts)
			return
		}
	}

	logging.Debug("Device %s is now ready to send messages", device.EUI)
	for i := 0; i < b.Config.DeviceMessages; i++ {
		if err := remoteDevice.SendMessageWithGenerator(generator); err != nil {
			logging.Warning("Device %s got error sending message #%d", device.EUI, i)
		}
		randomOffset := rand.Intn(b.Config.TransmissionDelay/10) - (b.Config.TransmissionDelay / 5)
		logging.Debug("Device %s has sent message %d of %d", device.EUI, i, b.Config.DeviceMessages)
		time.Sleep(time.Duration(b.Config.TransmissionDelay+randomOffset) * time.Millisecond)
	}
	logging.Info("Device %s has completed", device.EUI)
}

// Run the device emulation
func (b *BatchMode) Run(outgoingMessages chan string, publisher *EventRouter, app lassie.Application, gw lassie.Gateway) {
	b.OutgoingMessages = outgoingMessages
	b.Publisher = publisher
	b.Application = app

	// Power up our simulated devices
	completeWg := &sync.WaitGroup{}
	completeWg.Add(len(b.devices))
	for _, dev := range b.devices {
		go b.launchDevice(dev, completeWg)
	}
	logging.Info("....waiting for %d devices to complete", len(b.devices))
	completeWg.Wait()
}

// Failed returns true if the mode has failed
func (b *BatchMode) Failed() bool {
	return false
}
