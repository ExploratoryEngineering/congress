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

	"github.com/ExploratoryEngineering/congress/storage"
)

type memoryKeySequenceStorage struct {
	latencyStorage
	mutex     *sync.Mutex
	sequences map[string]uint64
}

func (m *memoryKeySequenceStorage) AllocateKeys(identifier string, interval uint64, initial uint64) (chan uint64, error) {
	m.RandomDelay()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	val, exists := m.sequences[identifier]
	if !exists {
		val = initial
	}
	m.sequences[identifier] = (val + interval)

	ret := make(chan uint64)
	go func() {
		counter := uint64(0)
		for counter < interval {
			ret <- (val + counter)
			counter++
		}
		close(ret)
	}()
	return ret, nil
}

func (m *memoryKeySequenceStorage) Close() {
	// nothing
}

// NewMemoryKeySequenceStorage returns a memory-backed KeySequenceStorage.
func NewMemoryKeySequenceStorage(minLatencyMs, maxLatencyMs int) storage.KeySequenceStorage {
	return &memoryKeySequenceStorage{latencyStorage{minLatencyMs, maxLatencyMs}, &sync.Mutex{}, make(map[string]uint64)}
}
