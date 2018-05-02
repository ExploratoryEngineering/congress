package memstore

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
	"sync"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
)

// memoryDeviceStorage is a (simple) memory-only implementation of the device storage interface
type memoryDeviceStorage struct {
	latencyStorage
	devices map[protocol.EUI]model.Device
	mutex   *sync.Mutex
}

// NewMemoryDeviceStorage creates a new MemoryDeviceStorage instance
func NewMemoryDeviceStorage(minLatencyMs, maxLatencyMs int) storage.DeviceStorage {
	return &memoryDeviceStorage{latencyStorage{minLatencyMs, maxLatencyMs}, make(map[protocol.EUI]model.Device), &sync.Mutex{}}
}

// GetByDevAddr returns the device with the matching device address
func (m *memoryDeviceStorage) GetByEUI(devEUI protocol.EUI) (model.Device, error) {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	dev, exists := m.devices[devEUI]
	if !exists {
		return model.Device{}, storage.ErrNotFound
	}
	return dev, nil
}

// Send elements in the list to a channel
func deviceSendFunc(list []model.Device, ch chan model.Device) {
	for _, val := range list {
		ch <- val
	}
	close(ch)
}

func (m *memoryDeviceStorage) GetByDevAddr(devAddr protocol.DevAddr) (chan model.Device, error) {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var ret []model.Device
	for _, device := range m.devices {
		if device.DevAddr == devAddr {
			ret = append(ret, device)
		}
	}
	sendCh := make(chan model.Device)
	go deviceSendFunc(ret, sendCh)
	return sendCh, nil
}

// GetByApplicationEUI returns the devices associated with the specified application
func (m *memoryDeviceStorage) GetByApplicationEUI(appEUI protocol.EUI) (chan model.Device, error) {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var ret []model.Device
	for _, device := range m.devices {
		if device.AppEUI == appEUI {
			ret = append(ret, device)
		}
	}
	sendCh := make(chan model.Device)
	go deviceSendFunc(ret, sendCh)
	return sendCh, nil
}

// Put adds a new device to the list of known devices. A device can be added only once.
func (m *memoryDeviceStorage) Put(device model.Device, appEUI protocol.EUI) error {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.devices[device.DeviceEUI]
	if exists {
		return storage.ErrAlreadyExists
	}
	device.AppEUI = appEUI
	m.devices[device.DeviceEUI] = device
	return nil
}

// AddDevNonce adds a new nonce to the device history
func (m *memoryDeviceStorage) AddDevNonce(device model.Device, nonce uint16) error {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing, exists := m.devices[device.DeviceEUI]
	if !exists {
		return storage.ErrNotFound
	}

	existing.DevNonceHistory = append(existing.DevNonceHistory, nonce)
	m.devices[device.DeviceEUI] = existing
	return nil

}

func (m *memoryDeviceStorage) UpdateState(device model.Device) error {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	existing, exists := m.devices[device.DeviceEUI]
	if !exists {
		return storage.ErrNotFound
	}
	existing.FCntDn = device.FCntDn
	existing.FCntUp = device.FCntUp
	existing.KeyWarning = device.KeyWarning
	m.devices[device.DeviceEUI] = existing
	return nil
}

func (m *memoryDeviceStorage) Delete(eui protocol.EUI) error {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, exists := m.devices[eui]
	if !exists {
		return storage.ErrNotFound
	}
	delete(m.devices, eui)
	return nil
}

// Close releases all of the allocated resources
func (m *memoryDeviceStorage) Close() {
	// nothing
}

func (m *memoryDeviceStorage) Update(device model.Device) error {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	existingDevice, exists := m.devices[device.DeviceEUI]
	if !exists {
		return storage.ErrNotFound
	}

	existingDevice.AppSKey = device.AppSKey
	existingDevice.NwkSKey = device.NwkSKey
	existingDevice.DevAddr = device.DevAddr
	existingDevice.FCntDn = device.FCntDn
	existingDevice.FCntUp = device.FCntUp
	existingDevice.RelaxedCounter = device.RelaxedCounter
	existingDevice.Tags = device.Tags
	m.devices[existingDevice.DeviceEUI] = existingDevice
	return nil
}
