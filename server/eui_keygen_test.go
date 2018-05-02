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
	"sync"
	"testing"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage/memstore"
)

func TestSimpleKeygen(t *testing.T) {
	// Make a MA-L for the keys
	ma, _ := protocol.NewMA([]byte{1, 2, 3})

	// NetID
	netID := uint32(0)

	storage := memstore.NewMemoryKeySequenceStorage(0, 0)

	keygen, err := NewEUIKeyGenerator(ma, netID, storage)
	if err != nil {
		t.Fatal("Couldn't create key generator: ", err)
	}

	generatedAppEUIs := make([]protocol.EUI, 0)
	generatedDeviceEUIs := make([]protocol.EUI, 0)
	generatedGenericKeys := make([]uint64, 0)

	const numKeys int = 2000
	for i := 0; i < numKeys; i++ {
		newAppEUI, err := keygen.NewAppEUI()
		if err != nil {
			t.Fatal("Got error generating app EUI: ", err)
		}
		generatedAppEUIs = append(generatedAppEUIs, newAppEUI)
		newDeviceEUI, err := keygen.NewDeviceEUI()
		if err != nil {
			t.Fatal("Got error generating device EUI: ", err)
		}
		generatedDeviceEUIs = append(generatedDeviceEUIs, newDeviceEUI)
		generatedGenericKeys = append(generatedGenericKeys, keygen.NewID("generic"))
	}
	// Ensure there's no collisions
	for i := 0; i < numKeys; i++ {
		for j := 0; j < numKeys; j++ {
			if j == i {
				continue
			}
			if generatedAppEUIs[i] == generatedAppEUIs[j] {
				t.Fatalf("Identical app EUI for i = %d, j = %d: %s", i, j, generatedAppEUIs[i])
			}
			if generatedDeviceEUIs[i] == generatedDeviceEUIs[j] {
				t.Fatalf("Identical device EUI for i = %d, j = %d: %s", i, j, generatedDeviceEUIs[i])
			}
			if generatedGenericKeys[i] == generatedGenericKeys[j] {
				t.Fatalf("Identical ID for i = %d, j = %d: %d", i, j, generatedGenericKeys[i])
			}
		}
	}
}

// Ensure you can't generate EUIs with MA-S and a NetID > 0x0F, MA-M with
// NetID > 0x0FFF or MA-L with NetID > 0xFFFF (we can't guarantee uniqueness)
func TestKeygenWithTooLargeNetID(t *testing.T) {
	storage := memstore.NewMemoryKeySequenceStorage(0, 0)

	// 25 bytes are used for the ID
	// MA-S is 36 bits, 64-36-25=3 bits for NetID
	maSmall, err := protocol.NewMA([]byte{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatal("Could not create MA-S EUI: ", err)
	}

	_, err = NewEUIKeyGenerator(maSmall, protocol.MaxNetworkBitsMAS+1, storage)
	if err == nil {
		t.Error("Expected error when creating key generator with too big NetID and MA-S EUI")
	}
	// Then something that will fit exactly
	_, err = NewEUIKeyGenerator(maSmall, protocol.MaxNetworkBitsMAS, storage)
	if err != nil {
		t.Error("Couldn't create key generator with MA-S/3 bit NetID: ", err)
	}

	// MA-M is 28 bits, 64-28-25=11 bits for NetID
	maMedium, err := protocol.NewMA([]byte{1, 2, 3, 4})
	if err != nil {
		t.Fatal("Could not create MA-M EUI: ", err)
	}

	// Create something with too big NetID
	_, err = NewEUIKeyGenerator(maMedium, protocol.MaxNetworkBitsMAM+1, storage)
	if err == nil {
		t.Error("Expected error when creating key generator with too big NetID and MA-M EUI")
	}
	// ...then something that will fit exactly
	_, err = NewEUIKeyGenerator(maMedium, protocol.MaxNetworkBitsMAM, storage)
	if err != nil {
		t.Error("Expected MA-M EIU and 11 bit NetID to fit: ", err)
	}

	// MA-L is 24 bits, 64-24-25=15 bits for NetID
	maLarge, err := protocol.NewMA([]byte{1, 2, 3})
	if err != nil {
		t.Fatal("Could not create MA-L EUI: ", err)
	}

	_, err = NewEUIKeyGenerator(maLarge, protocol.MaxNetworkBitsMAL+1, storage)
	if err == nil {
		t.Error("Expected error when using NetID > 15 bits")
	}
	_, err = NewEUIKeyGenerator(maLarge, protocol.MaxNetworkBitsMAL, storage)
	if err != nil {
		t.Error("Expected MA-L and 15 bit NetID to fit: ", err)
	}
}

// Ensure different NetIDs generate different EUIs for devices and applications
// even with the same sequence numbers
func TestKeygenWithDifferentNetID(t *testing.T) {
	storage := memstore.NewMemoryKeySequenceStorage(0, 0)
	// Use the same MA for both
	ma, _ := protocol.NewMA([]byte{0, 1, 2, 3, 4})
	keygen1, _ := NewEUIKeyGenerator(ma, 0, storage)
	keygen2, _ := NewEUIKeyGenerator(ma, 1, storage)

	eui1 := make([]protocol.EUI, 0)
	eui2 := make([]protocol.EUI, 0)

	keyCount := 1000
	for i := 0; i < keyCount; i++ {
		newEUI1, err := keygen1.NewDeviceEUI()
		if err != nil {
			t.Fatal("Got error generating EUI1: ", err)
		}
		eui1 = append(eui1, newEUI1)
		newEUI2, err := keygen2.NewDeviceEUI()
		if err != nil {
			t.Fatal("Got error generating EUI2: ", err)
		}
		eui2 = append(eui2, newEUI2)
	}

	for i := 0; i < keyCount; i++ {
		for j := 0; j < keyCount; j++ {
			if eui1[i] == eui2[j] {
				t.Errorf("Duplicate key for EUI1 (i=%d, j=%d). EUI=%s", i, j, eui1[i])
			}
		}
	}
}

type BigNumberSeq struct {
	mutex *sync.Mutex
}

func (f *BigNumberSeq) AllocateKeys(name string, interval uint64, initial uint64) (chan uint64, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	ret := make(chan uint64)
	go func() {
		for i := uint64(1); i < 1000000; i++ {
			ret <- uint64(maxID + i)
		}
		close(ret)
	}()
	return ret, nil
}
func (f *BigNumberSeq) Close() {
	// nah
}

// Ensure there's a panic when we risk duplicates. This is a tricky part. You
// normally have to either increase the NetID or change the EUI. If the IDs
// become too scarce it would be best to allocate a new MA block since the
// NetID is a scarce resource (and uniqueness for EUIs worldwide isn't stricly
// a requirement)
// Repeat the test with a new MA to ensure it won't leave permanent scars.
func TestAppEUIWithTooLargeID(t *testing.T) {

	storage := &BigNumberSeq{mutex: &sync.Mutex{}}
	// Use the same MA for both
	ma, _ := protocol.NewMA([]byte{0, 1, 2, 3, 4})
	keygen, _ := NewEUIKeyGenerator(ma, 0, storage)

	// This will fail with a panic
	_, err := keygen.NewAppEUI()
	if err == nil {
		t.Fatal("Did not get an error. Expected that.")
	}
}

func TestDevEUIWithTooLargeID(t *testing.T) {

	storage := &BigNumberSeq{mutex: &sync.Mutex{}}
	// Use the same MA for both
	ma, _ := protocol.NewMA([]byte{0, 1, 2, 3, 4})
	keygen, _ := NewEUIKeyGenerator(ma, 0, storage)

	// This will fail with a panic
	_, err := keygen.NewDeviceEUI()
	if err == nil {
		t.Fatal("Did not get an error. Expected that")
	}
}

type FailingSeq struct {
	fails    int
	maxfails int
	mutex    *sync.Mutex
}

func (f *FailingSeq) AllocateKeys(name string, interval uint64, initial uint64) (chan uint64, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.fails < f.maxfails {
		f.fails++
		return nil, errors.New("unable to allocate keys")
	}
	f.fails = 0
	ret := make(chan uint64)
	go func() {
		for i := 0; i < 10; i++ {
			ret <- uint64(i)
		}
		close(ret)
	}()
	return ret, nil
}
func (f *FailingSeq) Close() {
	// nah
}

// Make sure there's a few retries for a sequence
func TestFailingSequence(t *testing.T) {
	storage := &FailingSeq{fails: 0, maxfails: 2, mutex: &sync.Mutex{}}

	ma, _ := protocol.NewMA([]byte{0, 1, 2})
	keygen, _ := NewEUIKeyGenerator(ma, 0, storage)

	for i := 0; i < 30; i++ {
		keygen.NewDeviceEUI()
	}
}

// Ensure keys are allocated in a lazy fashion, ie they won't be allocated
// until someone retrieves a key.
func TestLazyInvocation(t *testing.T) {
	ksStorage := memstore.NewMemoryKeySequenceStorage(0, 0)

	ma, _ := protocol.NewMA([]byte{0, 1, 2, 3, 4})
	netID := uint32(0)
	identifier := "neteui"
	// Start by allocating keys directly
	name := fmt.Sprintf("%s/%04x/%s", ma.String(), netID, identifier)
	ch, err := ksStorage.AllocateKeys(name, 1, 1)
	if err != nil {
		t.Fatal("Got error allocating keys (first time)")
	}
	firstID := uint64(0)
	for v := range ch {
		firstID = v
	}
	kg1, _ := NewEUIKeyGenerator(ma, netID, ksStorage)
	kg2, _ := NewEUIKeyGenerator(ma, netID, ksStorage)

	// This ensures the keygen has started all of its goroutines
	kg1.NewID("foo1")
	kg2.NewID("foo2")

	// Repeat allocation. The ID should be the next in sequence
	ch, err = ksStorage.AllocateKeys(name, 1, 1)
	if err != nil {
		t.Fatal("Got error allocating keys (second time)")
	}
	lastID := uint64(0)
	for v := range ch {
		lastID = v
	}

	if lastID != (firstID + 1) {
		t.Fatalf("Expected next sequence to start at %d but it started at %d", (firstID + 1), lastID)
	}
}

// Simple benchmark for key generator. Grab 10 keys at a time
func BenchmarkKeygen(b *testing.B) {
	seqStorage := memstore.NewMemoryKeySequenceStorage(0, 0)
	ma, _ := protocol.NewMA([]byte{5, 4, 3, 2, 1})
	netID := uint32(1)
	memkeyGenerator, _ := NewEUIKeyGenerator(ma, netID, seqStorage)

	for i := 0; i < b.N; i++ {
		_, err := memkeyGenerator.NewAppEUI()
		if err != nil {
			b.Fatal("Got error retrieving key: ", err)
		}
	}
}
