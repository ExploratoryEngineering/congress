package storagetest

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
	"crypto/rand"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
)

// This is a set of default tests for storage backends.

func makeRandomEUI() protocol.EUI {
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	ret := protocol.EUI{}
	copy(ret.Octets[:], randomBytes)
	return ret
}

func makeRandomData() []byte {
	randomBytes := make([]byte, 30)
	rand.Read(randomBytes)
	return randomBytes
}

func makeRandomKey() protocol.AESKey {
	var keyBytes [16]byte
	copy(keyBytes[:], makeRandomData())
	return protocol.AESKey{Key: keyBytes}
}

func newRandomUser(id int) model.User {
	return model.User{
		ID:    model.UserID(fmt.Sprintf("id:%d", id)),
		Name:  fmt.Sprintf("User # %d", id),
		Email: fmt.Sprintf("random%d@example.com", id),
	}
}

var key = uint64(0)

// Create a temporary user in the database
func storeUser(id model.UserID, userManagement storage.UserManagement, t *testing.T) model.User {
	user := model.User{
		ID:    id,
		Email: "johndoe@example.com",
		Name:  "John Doe",
	}
	keyFunc := func(name string) uint64 {
		key++
		return key
	}
	if err := userManagement.AddOrUpdateUser(user, keyFunc); err != nil {
		t.Fatal("Got error creating user: ", err)
	}
	return user
}

// DoStorageTests tests all of the storage interfaces
func DoStorageTests(storageCollection *storage.Storage, t *testing.T) {
	if storageCollection.Application == nil {
		t.Fatal("Missing application storage")
	}
	if storageCollection.AppOutput == nil {
		t.Fatal("Missing App output storage")
	}
	if storageCollection.Device == nil {
		t.Fatal("Missing device storage")
	}
	if storageCollection.DeviceData == nil {
		t.Fatal("Missing device data storage")
	}
	if storageCollection.Gateway == nil {
		t.Fatal("Missing gateway storage")
	}
	if storageCollection.Sequence == nil {
		t.Fatal("Missing sequence storage")
	}
	if storageCollection.Token == nil {
		t.Fatal("Missing token storage")
	}
	if storageCollection.UserManagement == nil {
		t.Fatal("Missing user management storage")
	}

	userID := model.UserID("01")
	storeUser(userID, storageCollection.UserManagement, t)
	// Make another user for the tests
	storeUser(model.UserID("foo"), storageCollection.UserManagement, t)

	testApplicationStorage(storageCollection.Application, userID, t)
	testDeviceStorage(storageCollection.Application, storageCollection.Device, userID, t)
	testDataStorage(storageCollection.Application, storageCollection.Device, storageCollection.DeviceData, userID, t)
	testGatewayStorage(storageCollection.Gateway, userID, t)
	testTokenStorage(storageCollection.Token, userID, t)

	testSimpleKeySequence(storageCollection.Sequence, t)
	testMultipleSequences(storageCollection.Sequence, t)
	testConcurrentSequences(storageCollection.Sequence, t)
	testOutputStorage(storageCollection, t)
	testDownstreamStorage(storageCollection, t)

	testMultipleOpenClose(storageCollection.UserManagement, storageCollection.Gateway, t)
}

// testMultipleOpenClose ensures that rows, statements and connections aren't
// left open. Exercise the user management and gateway interfaces with
// multiple put and gets
func testMultipleOpenClose(userManagement storage.UserManagement, gateway storage.GatewayStorage, t *testing.T) {
	// Exec
	var id uint64 = 1000
	mutex := sync.Mutex{}
	keySeqFunc := func(name string) uint64 {
		mutex.Lock()
		defer mutex.Unlock()
		id = id + 1
		return id
	}

	const count = 1000
	// Create, then retrieve 1000 users
	for i := 0; i < count; i++ {
		user := newRandomUser(i)
		if err := userManagement.AddOrUpdateUser(user, keySeqFunc); err != nil {
			t.Fatalf("Got error adding or updating user with ID %s: %v", string(user.ID), err)
		}
		newGateway := model.Gateway{
			GatewayEUI: protocol.EUIFromUint64(uint64(i)),
			IP:         net.ParseIP("127.0.0.1"),
			StrictIP:   false,
			Tags:       model.NewTags(),
			Latitude:   float32(i),
			Longitude:  float32(i),
			Altitude:   float32(i),
		}
		if err := gateway.Put(newGateway, user.ID); err != nil {
			t.Fatalf("Got error storing gateway with EUI %s: %v", newGateway.GatewayEUI, err)
		}
	}
	for i := 0; i < count; i++ {
		user := newRandomUser(i)
		if err := userManagement.AddOrUpdateUser(user, keySeqFunc); err != nil {
			t.Fatalf("Got error adding or updating user: %v", err)
		}
		if _, err := gateway.Get(protocol.EUIFromUint64(uint64(i)), user.ID); err != nil {
			t.Fatalf("Could not retrieve gateway: %v", err)
		}
	}
}
