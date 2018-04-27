package server

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
	"errors"
	"sync"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/logging"
)

// --------------------------------------------------------------------------
// Interfaces used by the output manager
//
// The storage interface used by the output manager
type outputStorage interface {
	ListAll() (<-chan model.AppOutput, error)
}

// Event routing interface used by the app output manager
type router interface {
	Publish(key interface{}, msg interface{})
	Subscribe(key interface{}) <-chan interface{}
	Unsubscribe(msg <-chan interface{})
}

// --------------------------------------------------------------------------

// AppOutputManager is a memory-backed list of outputs. This includes all app
// outputs for the instance. The list of dispatchers are keyed on application
// EUI and output EUI.
//
type AppOutputManager struct {
	mutex       *sync.Mutex                              // To keep everything in sync
	dispatchers map[string]map[string]*messageDispatcher // A map keyed on app EUI, each containing a list of outputs
	eventRouter router
}

// NewAppOutputManager builds a new output manager.
func NewAppOutputManager(router router) *AppOutputManager {
	return &AppOutputManager{
		mutex:       &sync.Mutex{},
		dispatchers: make(map[string]map[string]*messageDispatcher),
		eventRouter: router,
	}
}

var (
	// ErrInvalidTransport is returned when the transport config is invalid (or unknown)
	ErrInvalidTransport = errors.New("invalid transport config")

	// ErrNotFound is returned when the output can't be found
	ErrNotFound = errors.New("output not found")
)

// LoadOutputs loads and starts all outputs from the storage
func (m *AppOutputManager) LoadOutputs(opstorage outputStorage) {
	outputs, err := opstorage.ListAll()
	if err != nil {
		logging.Error("Unable to load output list: %v", err)
		return
	}
	count := 0
	failed := 0
	for v := range outputs {
		if err := m.launchDispatcher(&v); err != nil {
			logging.Warning("Unable to launch output with EUI %s: %v", v.EUI, err)
			failed++
			continue
		}
		count++
	}
	logging.Info("%d dispatchers launched for outputs, %d failed", count, failed)
}

// Shutdown closes all running message dispatchers
func (m *AppOutputManager) Shutdown() {
	count := 0
	for _, outputMap := range m.dispatchers {
		for _, v := range outputMap {
			m.stopDispatcher(v.output())
			count++
		}
	}
	logging.Info("%d dispatchers for outputs shut down", count)
}

// Launch a new message dispatcher. If the dispatcher is already running it will
// be stopped and restarted with the new configuration. The logs and existing
// message channel is reused for the new dispatchers
func (m *AppOutputManager) launchDispatcher(op *model.AppOutput) error {
	// Try to get the transport
	transport := getTransport(op)
	if transport == nil {
		return ErrInvalidTransport
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	list, ok := m.dispatchers[op.AppEUI.String()]
	if !ok {
		list = make(map[string]*messageDispatcher)
	}

	existing, ok := list[op.EUI.String()]
	if ok {
		// Stop and replace with updated app output config
		logging.Debug("Updating dispatcher with EUI %s", op.EUI)
		existing.stop()
		existing = newMessageDispatcher(op, existing.logs(), existing.messageChannel(), transport)
	} else {
		// Launch a new message dispatcher
		logging.Debug("Subscribing to %s for output %s", op.AppEUI, op.EUI)
		messages := m.eventRouter.Subscribe(op.AppEUI)
		ml := NewMemoryLogger()
		new := newMessageDispatcher(op, &ml, messages, transport)
		list[op.EUI.String()] = new
		existing = new
	}
	list[op.EUI.String()] = existing
	m.dispatchers[op.AppEUI.String()] = list
	existing.start()
	return nil
}

// Add adds a new app output
func (m *AppOutputManager) Add(op *model.AppOutput) error {
	return m.launchDispatcher(op)
}

// Update updates and restarts the app output
func (m *AppOutputManager) Update(op *model.AppOutput) error {
	return m.launchDispatcher(op)
}

// stopDispatcher stops the dispatcher. It will release the resources used
// by the dispatcher
func (m *AppOutputManager) stopDispatcher(op *model.AppOutput) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	list, ok := m.dispatchers[op.AppEUI.String()]
	if !ok {
		return ErrNotFound
	}
	existing, ok := list[op.EUI.String()]
	if !ok {
		return ErrNotFound
	}
	existing.stop()
	m.eventRouter.Unsubscribe(existing.messageChannel())
	delete(list, op.EUI.String())
	m.dispatchers[op.AppEUI.String()] = list
	return nil
}

// Remove removes the output and stops it
func (m *AppOutputManager) Remove(op *model.AppOutput) error {
	return m.stopDispatcher(op)
}

// GetStatusAndLogs returns the status and logs for the specified app output
func (m *AppOutputManager) GetStatusAndLogs(op *model.AppOutput) (string, *MemoryLogger, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	outputs, ok := m.dispatchers[op.AppEUI.String()]
	if !ok {
		return "", nil, ErrNotFound
	}
	dispatcher, ok := outputs[op.EUI.String()]
	if !ok {
		return "", nil, ErrNotFound
	}
	return dispatcher.status(), dispatcher.logs(), nil
}
