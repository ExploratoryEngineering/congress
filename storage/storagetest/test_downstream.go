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
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
)

func testDownstreamStorage(s *storage.Storage, t *testing.T) {
	application := model.NewApplication()
	application.AppEUI = makeRandomEUI()
	s.Application.Put(application, model.SystemUserID)

	testDevice := model.NewDevice()
	testDevice.AppEUI = application.AppEUI
	testDevice.DeviceEUI = makeRandomEUI()
	testDevice.AppSKey = makeRandomKey()
	testDevice.DevAddr = protocol.DevAddrFromUint32(0x01020304)
	testDevice.NwkSKey = makeRandomKey()
	s.Device.Put(testDevice, application.AppEUI)

	downstreamMsg := model.NewDownstreamMessage(testDevice.DeviceEUI, 42)
	downstreamMsg.Ack = false
	downstreamMsg.Data = "aabbccddeeff"
	if err := s.DeviceData.PutDownstream(testDevice.DeviceEUI, downstreamMsg); err != nil {
		t.Fatal("Couldn't store downstream message: ", err)
	}

	newDownstreamMsg := model.NewDownstreamMessage(testDevice.DeviceEUI, 43)
	newDownstreamMsg.Ack = false
	newDownstreamMsg.Data = "aabbccddeeff"
	if err := s.DeviceData.PutDownstream(testDevice.DeviceEUI, newDownstreamMsg); err == nil {
		t.Fatal("Shouldn't be able to store another downstream message")
	}

	if err := s.DeviceData.DeleteDownstream(testDevice.DeviceEUI); err != nil {
		t.Fatalf("Couldn't remove downstream message: %v", err)
	}

	if err := s.DeviceData.DeleteDownstream(testDevice.DeviceEUI); err != storage.ErrNotFound {
		t.Fatalf("Should get ErrNotFound when removing message but got: %v", err)
	}

	if _, err := s.DeviceData.GetDownstream(testDevice.DeviceEUI); err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound but got %v", err)
	}

	if err := s.DeviceData.PutDownstream(testDevice.DeviceEUI, newDownstreamMsg); err != nil {
		t.Fatalf("Should be able to store the new downstream message but got %v: ", err)
	}

	time2 := time.Now().Unix()
	if err := s.DeviceData.UpdateDownstream(testDevice.DeviceEUI, time2, 0); err != nil {
		t.Fatal("Should be able to update sent time but got error: ", err)
	}

	newDownstreamMsg.SentTime = time2
	stored, err := s.DeviceData.GetDownstream(testDevice.DeviceEUI)
	if err != nil {
		t.Fatal("Got error retrieving downstream message: ", err)
	}
	if stored != newDownstreamMsg {
		t.Fatalf("Sent time isn't updated properly. Got %+v but expected %+v", stored, newDownstreamMsg)
	}

	time3 := time.Now().Unix()
	if err := s.DeviceData.UpdateDownstream(testDevice.DeviceEUI, 0, time3); err != nil {
		t.Fatal("Got error updating downstream message: ", err)
	}

	stored, err = s.DeviceData.GetDownstream(testDevice.DeviceEUI)
	if err != nil {
		t.Fatal("Got error retrieving downstream message: ", err)
	}
	if stored.AckTime != time3 {
		t.Fatalf("Ack time isn't updated properly. Got %d but expected %d", stored.AckTime, time3)
	}

	if err := s.DeviceData.DeleteDownstream(testDevice.DeviceEUI); err != nil {
		t.Fatalf("Did not expect error when deleting downstream but got %v", err)
	}

	if err := s.DeviceData.UpdateDownstream(testDevice.DeviceEUI, 0, 0); err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound when updating nonexisting message but got %v", err)
	}

}
