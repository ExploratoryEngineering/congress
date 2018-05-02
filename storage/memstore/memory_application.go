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

type appUser struct {
	app    model.Application
	userID model.UserID
}

// memoryApplicationStorage is a memory implementation of the ApplicationStorage implementation
type memoryApplicationStorage struct {
	applications map[protocol.EUI]appUser
	mutex        *sync.Mutex
	devices      storage.DeviceStorage
}

// NewMemoryApplicationStorage returns a new instance of MemoryApplicationStorage
func NewMemoryApplicationStorage(deviceStorage storage.DeviceStorage) storage.ApplicationStorage {
	ret := memoryApplicationStorage{
		applications: make(map[protocol.EUI]appUser),
		mutex:        &sync.Mutex{},
		devices:      deviceStorage,
	}
	return &ret
}

// Put stores a new application
func (m *memoryApplicationStorage) Put(application model.Application, userID model.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.applications[application.AppEUI]
	if exists {
		return storage.ErrAlreadyExists
	}
	m.applications[application.AppEUI] = appUser{application, userID}
	return nil
}

// GetByEUI returns the application that matches the specified EUI
func (m *memoryApplicationStorage) GetByEUI(eui protocol.EUI, userID model.UserID) (model.Application, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	app, exists := m.applications[eui]
	if !exists || app.userID != userID {
		return model.Application{}, storage.ErrNotFound
	}
	return app.app, nil
}

func appSendFunc(list []model.Application, ch chan model.Application) {
	for _, val := range list {
		ch <- val
	}
	close(ch)
}

// GetByNetworkEUI returns a channel with all applications in a given network
func (m *memoryApplicationStorage) GetList(userID model.UserID) (chan model.Application, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var ret []model.Application
	for _, app := range m.applications {
		if app.userID == userID {
			ret = append(ret, app.app)
		}
	}
	sendCh := make(chan model.Application)
	go appSendFunc(ret, sendCh)
	return sendCh, nil
}

func (m *memoryApplicationStorage) Delete(eui protocol.EUI, userID model.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing, exists := m.applications[eui]
	if !exists || existing.userID != userID {
		return storage.ErrNotFound
	}

	devices, err := m.devices.GetByApplicationEUI(eui)
	if err != nil {
		return err
	}
	for range devices {
		return storage.ErrDeleteConstraint
	}

	delete(m.applications, eui)
	return nil
}

func (m *memoryApplicationStorage) Update(application model.Application, userID model.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	app, exists := m.applications[application.AppEUI]
	if !exists || app.userID != userID {
		return storage.ErrNotFound
	}
	app.app.Tags = application.Tags
	m.applications[application.AppEUI] = app
	return nil
}

// Close releases all of the allocated resources
func (m *memoryApplicationStorage) Close() {
	// nothing
}
