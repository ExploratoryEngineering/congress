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

	"time"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

// MemoryOutput is a memory-backed output storage
type MemoryOutput struct {
	mutex   *sync.Mutex
	outputs map[protocol.EUI][]model.AppOutput
}

// NewMemoryOutput creates a new memory output
func NewMemoryOutput() storage.AppOutputStorage {
	return &MemoryOutput{
		&sync.Mutex{},
		make(map[protocol.EUI][]model.AppOutput),
	}
}

// Put stores a new output
func (m *MemoryOutput) Put(newOutput model.AppOutput) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	list, ok := m.outputs[newOutput.AppEUI]
	if !ok {
		list = make([]model.AppOutput, 0)
	}
	for _, v := range list {
		if v.EUI == newOutput.EUI {
			return storage.ErrAlreadyExists
		}
	}
	list = append(list, newOutput)
	m.outputs[newOutput.AppEUI] = list
	return nil
}

// Update updates the configuration for the output.
func (m *MemoryOutput) Update(output model.AppOutput) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	list, ok := m.outputs[output.AppEUI]
	if !ok {
		return storage.ErrNotFound
	}

	for i, v := range list {
		if v.EUI == output.EUI {
			list[i] = output
			m.outputs[output.AppEUI] = list
			return nil
		}
	}

	return storage.ErrNotFound
}

// Delete removes the output. Outputs are also removed automatically when
// an application is removed
func (m *MemoryOutput) Delete(output model.AppOutput) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	list, ok := m.outputs[output.AppEUI]
	if !ok {
		return storage.ErrNotFound
	}

	for i, v := range list {
		if v.EUI == output.EUI {
			list = append(list[0:i], list[i+1:]...)
			m.outputs[output.AppEUI] = list
			return nil
		}
	}

	return storage.ErrNotFound
}

// GetByApplication returns outputs for a single application
func (m *MemoryOutput) GetByApplication(appEUI protocol.EUI) (<-chan model.AppOutput, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	list, ok := m.outputs[appEUI]
	if !ok {
		return nil, storage.ErrNotFound
	}

	ret := make(chan model.AppOutput, 20)
	retList := make([]model.AppOutput, 0)
	retList = append(retList, list...)
	go func() {
		defer close(ret)
		for _, v := range retList {
			select {
			case ret <- v:
			case <-time.After(1 * time.Second):
				logging.Info("Dumping channel when it hasn't been read for 1 s")
				return
			}
		}
	}()
	return ret, nil
}

// ListAll lists all of the outputs
func (m *MemoryOutput) ListAll() (<-chan model.AppOutput, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ret := make(chan model.AppOutput, 20)
	list := make([]model.AppOutput, 0)
	for _, v := range m.outputs {
		list = append(list, v...)
	}

	go func() {
		defer close(ret)
		for _, v := range list {
			select {
			case ret <- v:
			case <-time.After(1 * time.Second):
				logging.Info("Dumping channel when it hasn't been read for 1 s")
				return
			}
		}
	}()
	return ret, nil
}
