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
	"github.com/ExploratoryEngineering/congress/storage"
)

type memoryUserManagement struct {
	mutex  *sync.Mutex
	owners map[model.UserID]uint64
}

func (m *memoryUserManagement) AddOrUpdateUser(user model.User, keyFunc storage.KeyGeneratorFunc) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ownerID, exists := m.owners[user.ID]
	if !exists {
		ownerID = keyFunc("connectUserOwner")
	}
	m.owners[user.ID] = ownerID
	return nil
}

func (m *memoryUserManagement) GetOwner(userID model.UserID) (uint64, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	ownerID, exists := m.owners[userID]
	if !exists {
		return 0, storage.ErrNotFound
	}
	return ownerID, nil
}

func (m *memoryUserManagement) Close() {
	// empty
}

// NewMemoryUserManagement returns a memory-backed UserManagement type
func NewMemoryUserManagement() storage.UserManagement {
	return &memoryUserManagement{
		mutex:  &sync.Mutex{},
		owners: make(map[model.UserID]uint64),
	}
}
