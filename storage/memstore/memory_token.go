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

type memToken struct {
	token  model.APIToken
	userID model.UserID
}

// memoryTokenStorage is a memory-based implementation of TokenStorage
type memoryTokenStorage struct {
	Tokens map[string]memToken
	mutex  *sync.Mutex
}

// NewMemoryTokenStorage creates a new storage.TokenStorage implementation that
// stores everything in memory.
func NewMemoryTokenStorage() storage.TokenStorage {
	return &memoryTokenStorage{
		Tokens: make(map[string]memToken),
		mutex:  &sync.Mutex{},
	}
}

func (m *memoryTokenStorage) Put(token model.APIToken, userID model.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.Tokens[token.Token]
	if exists {
		return storage.ErrAlreadyExists
	}

	m.Tokens[token.Token] = memToken{token, userID}
	return nil
}

func (m *memoryTokenStorage) Get(token string) (model.APIToken, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	apiToken, exists := m.Tokens[token]
	if !exists {
		return apiToken.token, storage.ErrNotFound
	}
	return apiToken.token, nil
}

func (m *memoryTokenStorage) GetList(userID model.UserID) (chan model.APIToken, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	returnList := make([]model.APIToken, 0)
	for _, v := range m.Tokens {
		if v.userID == userID {
			returnList = append(returnList, v.token)
		}
	}
	returnChannel := make(chan model.APIToken)
	go func() {
		for _, v := range returnList {
			returnChannel <- v
		}
		close(returnChannel)
	}()
	return returnChannel, nil
}

func (m *memoryTokenStorage) Delete(token string, userID model.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing, exists := m.Tokens[token]
	if !exists || existing.userID != userID {
		return storage.ErrNotFound
	}
	delete(m.Tokens, token)
	return nil
}

func (m *memoryTokenStorage) Update(token model.APIToken, userID model.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing, exists := m.Tokens[token.Token]
	if !exists || existing.userID != userID {
		return storage.ErrNotFound
	}
	m.Tokens[token.Token] = memToken{token, userID}
	return nil
}

func (m *memoryTokenStorage) Close() {
	// nothing to do
}
