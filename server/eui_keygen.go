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
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

const (
	maxID uint64 = (1 << 26) - 1
)

// The key dispatcher is responsible for handing out keys for a single
// identifier
type keyDispatcher struct {
	response    chan uint64
	identifier  string
	keySequence chan uint64
	keyStorage  storage.KeySequenceStorage
	interval    uint64
	acquire     chan bool // aquire channel - signal for "new id requested"
}

// KeyGenerator generates unique application and device EUIs with the help of
// a MA, NetID and a storage-backed sequence. If the backend is down it will
// retry forever until the new sequence can be created. If the backend isn't
// responding and the allocated block is used the ID generator will block
// until a new block is allocated.
type KeyGenerator struct {
	ma                  protocol.MA
	netID               uint32
	keyStorage          storage.KeySequenceStorage
	appEUIdispatcher    keyDispatcher
	deviceEUIdispatcher keyDispatcher
	outputEUIdispatcher keyDispatcher
	mutex               *sync.Mutex
	sequences           map[string]*keyDispatcher
}

// NewAppEUI creates a new application EUI. This call might block if the storage
// backend is down. The returned EUI will always be a valid EUI but if the error
// field is set it won't be unique.
func (k *KeyGenerator) NewAppEUI() (protocol.EUI, error) {
	k.appEUIdispatcher.acquire <- true
	newID := <-k.appEUIdispatcher.response
	var err error
	if newID > maxID {
		err = errors.New("key space is exhausted for application EUI")
	}
	return protocol.NewApplicationEUI(k.ma, k.netID, uint32(newID&0xFFFFFFFF)), err
}

// NewDeviceEUI creates a new device EUI. This call might block if the storage
// backend is down. The returned EUI will always be a valid EUI but if the error
// field is set it won't be unique.
func (k *KeyGenerator) NewDeviceEUI() (protocol.EUI, error) {
	k.deviceEUIdispatcher.acquire <- true
	newID := <-k.deviceEUIdispatcher.response
	var err error
	if newID > maxID {
		err = errors.New("key space is exhausted for device EUI")
	}
	return protocol.NewDeviceEUI(k.ma, k.netID, uint32(newID&0xFFFFFFFF)), err
}

func newDispatcher(interval uint64, name string, keyStorage storage.KeySequenceStorage) keyDispatcher {
	return keyDispatcher{
		response:   make(chan uint64),
		identifier: name,
		keyStorage: keyStorage,
		interval:   interval,
		acquire:    make(chan bool),
	}
}

// NewID creates a new generic ID. This call might block if the storage
// backend is down. The returned EUI will always be a valid EUI but if the error
// field is set it won't be unique.
func (k *KeyGenerator) NewID(identifier string) uint64 {
	// This might need a new dispatcher.
	k.mutex.Lock()
	defer k.mutex.Unlock()
	seq, exists := k.sequences[identifier]
	if !exists {
		newDispatcher := newDispatcher(5, fmt.Sprintf("%s/%04x/%s", k.ma.String(), k.netID, identifier), k.keyStorage)
		seq = &newDispatcher
		k.sequences[identifier] = seq
		go seq.dispatch()
	}
	seq.acquire <- true
	return <-seq.response
}

// NewOutputEUI generates a new EUI for an output. It uses the same scope as
// application EUIs.
func (k *KeyGenerator) NewOutputEUI() (protocol.EUI, error) {
	k.outputEUIdispatcher.acquire <- true
	newID := <-k.outputEUIdispatcher.response
	var err error
	if newID > maxID {
		err = errors.New("key space is exhausted for output EUI")
	}
	return protocol.NewApplicationEUI(k.ma, k.netID, uint32(newID&0xFFFFFFFF)), err
}

func (d *keyDispatcher) dispatch() {
	for {
		<-d.acquire
		ok := false
		id := uint64(0)
		retries := int32(1)
		for !ok {
			for d.keySequence == nil {
				var err error
				d.keySequence, err = d.keyStorage.AllocateKeys(d.identifier, d.interval, 1)
				if err != nil {
					logging.Warning("Unable to allocate keys for sequence (attempt %d) %s: %v",
						retries, d.identifier, err)
					// Sleep for random time between retries. The upper bound is a function of
					// the number of retries.
					<-time.After(time.Duration(rand.Int31n(retries*500)) * time.Millisecond)
					retries++
				}
			}
			id, ok = <-d.keySequence
			if !ok {
				d.keySequence = nil
			}
		}
		d.response <- id
	}
}

// NewEUIKeyGenerator creates a new KeyGenerator instance
func NewEUIKeyGenerator(ma protocol.MA, netID uint32, keyStorage storage.KeySequenceStorage) (KeyGenerator, error) {
	switch ma.Size {
	case protocol.MALarge:
		if netID > 0x7FFF {
			return KeyGenerator{}, fmt.Errorf("MA-S EUIs cannot have NetID bigger than 15 bits (NetID=0x%X)", netID)
		}
	case protocol.MAMedium:
		if netID > 0x7FF {
			return KeyGenerator{}, fmt.Errorf("MA-S EUIs cannot have NetID bigger than 11 bits (NetID=0x%X)", netID)
		}
	default:
		if netID > 0x7 {
			return KeyGenerator{}, fmt.Errorf("MA-S EUIs cannot have NetID bigger than three bits (NetID=0x%X)", netID)
		}
	}
	ret := KeyGenerator{
		ma:                  ma,
		netID:               netID,
		keyStorage:          keyStorage,
		appEUIdispatcher:    newDispatcher(10, fmt.Sprintf("%s/%04x/appeui", ma.String(), netID), keyStorage),
		deviceEUIdispatcher: newDispatcher(100, fmt.Sprintf("%s/%04x/deveui", ma.String(), netID), keyStorage),
		outputEUIdispatcher: newDispatcher(10, fmt.Sprintf("%s/%04x/outputeui", ma.String(), netID), keyStorage),
		sequences:           make(map[string]*keyDispatcher),
		mutex:               &sync.Mutex{},
	}
	go ret.appEUIdispatcher.dispatch()
	go ret.deviceEUIdispatcher.dispatch()
	go ret.outputEUIdispatcher.dispatch()
	return ret, nil
}
