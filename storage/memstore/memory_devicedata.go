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
	"fmt"
	"sort"
	"sync"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
)

type memoryDataList map[int64]model.DeviceData
type memoryDeviceData map[protocol.EUI]memoryDataList
type downstreamMessages map[protocol.EUI]model.DownstreamMessage

// memoryDataStorage implements a backend storage
type memoryDataStorage struct {
	latencyStorage
	deviceData memoryDeviceData      // the data that the device stored
	mutex      *sync.Mutex           // because concurrent
	devStorage storage.DeviceStorage // Need this for lookups
	downstream downstreamMessages    // Downstream message store
}

// NewMemoryDataStorage makes a new MemoryDataStorage instance
func NewMemoryDataStorage(devStorage storage.DeviceStorage, minLatencyMs, maxLatencyMs int) storage.DataStorage {
	return &memoryDataStorage{latencyStorage{minLatencyMs, maxLatencyMs}, make(memoryDeviceData), &sync.Mutex{}, devStorage, make(downstreamMessages)}
}

// Put stores data from an end-device
func (m *memoryDataStorage) Put(deviceEUI protocol.EUI, data model.DeviceData) error {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var dataList = m.deviceData[deviceEUI]
	if dataList == nil {
		dataList = make(memoryDataList)
		m.deviceData[deviceEUI] = dataList
	}
	_, exists := dataList[data.Timestamp]
	if exists {
		return fmt.Errorf("a data item already exists for the timestamp %d", data.Timestamp)
	}
	data.DeviceEUI = deviceEUI
	dataList[data.Timestamp] = data
	return nil
}

// Send elements in the list to a channel
func dataSendFunc(list []model.DeviceData, ch chan model.DeviceData) {
	for _, val := range list {
		ch <- val
	}
	close(ch)
}

// GetByDeviceEUI returns data for a given device. The order is not guaranteed.
func (m *memoryDataStorage) GetByDeviceEUI(deviceEUI protocol.EUI, limit int) (chan model.DeviceData, error) {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	dataList, exists := m.deviceData[deviceEUI]
	if !exists {
		dataList = make(memoryDataList)
	}
	retCh := make(chan model.DeviceData)
	var ret []model.DeviceData
	for _, data := range dataList {
		ret = append(ret, data)
	}

	sort.Sort(byTimestamp(ret))
	if len(ret) > limit {
		ret = ret[:limit]
	}
	go dataSendFunc(ret, retCh)
	return retCh, nil
}

type byTimestamp []model.DeviceData

func (a byTimestamp) Len() int {
	return len(a)
}

func (a byTimestamp) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a byTimestamp) Less(i, j int) bool {
	return a[i].Timestamp < a[j].Timestamp
}

func (m *memoryDataStorage) GetByApplicationEUI(applicationEUI protocol.EUI, limit int) (chan model.DeviceData, error) {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	list, err := m.devStorage.GetByApplicationEUI(applicationEUI)
	if err != nil {
		return nil, err
	}

	fullDataList := make([]model.DeviceData, 0)
	for device := range list {
		dataList, exists := m.deviceData[device.DeviceEUI]
		if !exists {
			continue
		}
		for _, data := range dataList {
			fullDataList = append(fullDataList, data)
		}
	}
	sort.Sort(byTimestamp(fullDataList))
	retCh := make(chan model.DeviceData)
	if len(fullDataList) > limit {
		fullDataList = fullDataList[:limit]
	}
	go dataSendFunc(fullDataList, retCh)
	return retCh, nil
}

// Close releases all of the resources allocated by the MemoryDataStorage instance.
func (m *memoryDataStorage) Close() {
	// nothing
}

func (m *memoryDataStorage) deleteDeviceData(eui protocol.EUI) {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.deviceData, eui)
}

func (m *memoryDataStorage) PutDownstream(deviceEUI protocol.EUI, message model.DownstreamMessage) error {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.downstream[deviceEUI]
	if exists {
		return storage.ErrAlreadyExists
	}
	m.downstream[deviceEUI] = message
	return nil
}

func (m *memoryDataStorage) DeleteDownstream(deviceEUI protocol.EUI) error {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, exists := m.downstream[deviceEUI]
	if !exists {
		return storage.ErrNotFound
	}
	delete(m.downstream, deviceEUI)
	return nil
}

func (m *memoryDataStorage) GetDownstream(deviceEUI protocol.EUI) (model.DownstreamMessage, error) {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	existing, exists := m.downstream[deviceEUI]
	if !exists {
		return existing, storage.ErrNotFound
	}
	return existing, nil
}

func (m *memoryDataStorage) UpdateDownstream(deviceEUI protocol.EUI, sentTime int64, ackTime int64) error {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	existing, exists := m.downstream[deviceEUI]
	if !exists {
		return storage.ErrNotFound
	}
	existing.SentTime = sentTime
	existing.AckTime = ackTime
	m.downstream[deviceEUI] = existing
	return nil
}
