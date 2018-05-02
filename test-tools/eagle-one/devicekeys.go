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
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/telenordigital/lassie-go"
)

// DeviceKeys stores the keys, EUIs and DevAddr in Congress-friendly formats
type DeviceKeys struct {
	AppEUI  protocol.EUI
	DevEUI  protocol.EUI
	AppKey  protocol.AESKey
	AppSKey protocol.AESKey  // App session key
	NwkSKey protocol.AESKey  // Network session key
	DevAddr protocol.DevAddr // Device address
}

// NewDeviceKeys creates a new device key type from a Lassie Device
func NewDeviceKeys(appEUI string, device lassie.Device) (DeviceKeys, error) {
	d := DeviceKeys{}
	var err error
	if d.AppEUI, err = protocol.EUIFromString(appEUI); err != nil {
		return d, err
	}
	if d.DevEUI, err = protocol.EUIFromString(device.EUI); err != nil {
		return d, err
	}
	if d.AppKey, err = protocol.AESKeyFromString(device.ApplicationKey); err != nil {
		return d, err
	}
	if d.DevAddr, err = protocol.DevAddrFromString(device.DeviceAddress); err != nil {
		return d, err
	}
	if d.NwkSKey, err = protocol.AESKeyFromString(device.NetworkSessionKey); err != nil {
		return d, err
	}
	if d.AppSKey, err = protocol.AESKeyFromString(device.ApplicationSessionKey); err != nil {
		return d, err
	}
	return d, nil
}
