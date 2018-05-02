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

type memGateway struct {
	gw     model.Gateway
	userID model.UserID
}

// memoryGatewayStorage implements GatewayStorage
type memoryGatewayStorage struct {
	mutex    *sync.Mutex
	gateways map[protocol.EUI]memGateway
}

func (m *memoryGatewayStorage) Put(gateway model.Gateway, userID model.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.gateways[gateway.GatewayEUI]
	if exists {
		return storage.ErrAlreadyExists
	}

	m.gateways[gateway.GatewayEUI] = memGateway{gateway, userID}
	return nil
}

func (m *memoryGatewayStorage) Delete(eui protocol.EUI, userID model.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing, exists := m.gateways[eui]
	if !exists || existing.userID != userID {
		return storage.ErrNotFound
	}
	delete(m.gateways, eui)
	return nil
}

func (m *memoryGatewayStorage) GetList(userID model.UserID) (chan model.Gateway, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var gatewayList = make([]model.Gateway, 0)
	for _, val := range m.gateways {
		if val.userID == userID {
			gatewayList = append(gatewayList, val.gw)
		}
	}

	ret := make(chan model.Gateway)
	go func() {
		for _, v := range gatewayList {
			ret <- v
		}
		close(ret)
	}()
	return ret, nil
}

func (m *memoryGatewayStorage) ListAll() (chan model.PublicGatewayInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var gatewayList = make([]model.PublicGatewayInfo, 0)
	for _, val := range m.gateways {
		gatewayList = append(gatewayList, model.PublicGatewayInfo{
			EUI:       val.gw.GatewayEUI.String(),
			Latitude:  val.gw.Latitude,
			Longitude: val.gw.Longitude,
			Altitude:  val.gw.Altitude})
	}

	ret := make(chan model.PublicGatewayInfo)
	go func() {
		for _, v := range gatewayList {
			ret <- v
		}
		close(ret)
	}()
	return ret, nil
}

func (m *memoryGatewayStorage) Get(eui protocol.EUI, userID model.UserID) (model.Gateway, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	gw, exists := m.gateways[eui]
	if !exists || gw.userID != userID {
		return model.Gateway{}, storage.ErrNotFound
	}

	return gw.gw, nil
}

func (m *memoryGatewayStorage) Update(gateway model.Gateway, userID model.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing, exists := m.gateways[gateway.GatewayEUI]
	if !exists || existing.userID != userID {
		return storage.ErrNotFound
	}

	existing.gw.IP = gateway.IP
	existing.gw.Altitude = gateway.Altitude
	existing.gw.Latitude = gateway.Latitude
	existing.gw.Longitude = gateway.Longitude
	existing.gw.StrictIP = gateway.StrictIP
	existing.gw.Tags = gateway.Tags

	m.gateways[gateway.GatewayEUI] = existing

	return nil
}
func (m *memoryGatewayStorage) Close() {

}

// NewMemoryGatewayStorage returns a memory-backed gateway storage
func NewMemoryGatewayStorage() storage.GatewayStorage {
	return &memoryGatewayStorage{
		mutex:    &sync.Mutex{},
		gateways: make(map[protocol.EUI]memGateway),
	}
}
