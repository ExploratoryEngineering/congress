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

func testDataStorage(
	appStorage storage.ApplicationStorage,
	devStorage storage.DeviceStorage,
	dataStorage storage.DataStorage,
	userID model.UserID,
	t *testing.T) {

	app := model.Application{
		AppEUI: makeRandomEUI(),
		Tags:   model.NewTags(),
	}
	if err := appStorage.Put(app, userID); err != nil {
		t.Error("Got error adding application: ", err)
	}

	device := model.Device{
		DeviceEUI: makeRandomEUI(),
		AppEUI:    app.AppEUI,
		DevAddr: protocol.DevAddr{
			NwkID:   1,
			NwkAddr: 0x400004,
		},
		FCntUp: 4,
		Tags:   model.NewTags(),
	}

	err := devStorage.Put(device, app.AppEUI)
	if err != nil {
		t.Error("Error putting device: ", err)
	}

	data1 := makeRandomData()
	data2 := makeRandomData()

	deviceData1 := model.DeviceData{Timestamp: 1, Data: data1, DeviceEUI: device.DeviceEUI, Frequency: 1.0}
	deviceData2 := model.DeviceData{Timestamp: 2, Data: data2, DeviceEUI: device.DeviceEUI, Frequency: 2.0}

	if err = dataStorage.Put(device.DeviceEUI, deviceData1); err != nil {
		t.Error("Could not store data: ", err)
	}

	if err = dataStorage.Put(device.DeviceEUI, deviceData2); err != nil {
		t.Error("Could not store 2nd data: ", err)
	}

	// Storing it a 2nd time won't work
	if err = dataStorage.Put(device.DeviceEUI, deviceData1); err == nil {
		t.Error("Shouldn't be able to store data twice (data#1)")
	}

	if err = dataStorage.Put(device.DeviceEUI, deviceData2); err == nil {
		t.Error("Shouldn't be able to store data twice (data#2)")
	}

	// Test retrieval
	dataChan, err := dataStorage.GetByDeviceEUI(device.DeviceEUI, 2)
	if err != nil {
		t.Error("Did not expect error when retrieving data")
	}

	var firstData, secondData model.DeviceData
	// Read from channels, time out if there's no data
	timestamps := int64(0)
	select {
	case firstData = <-dataChan:
		timestamps += firstData.Timestamp
	case <-time.After(time.Second * 2):
		t.Error("Timed out waiting for data # 1")
	}

	select {
	case secondData = <-dataChan:
		timestamps += secondData.Timestamp
	case <-time.After(time.Second * 2):
		t.Error("Timed out waiting for data # 2")
	}

	if timestamps != int64(3) {
		t.Error("Did not get the correct data pieces")
	}

	// Try retrieving from device with no data.
	var dataChannel chan model.DeviceData
	if _, err = dataStorage.GetByDeviceEUI(makeRandomEUI(), 2); err != nil {
		t.Error("Did not expect error when retrieving from non-existing device")
	}
	select {
	case <-dataChannel:
		t.Fatal("Did not expect any data on channel")
	case <-time.After(100 * time.Millisecond):
		// This is OK
	}

	// Read application device data. Should be the same as the device data.
	appChan, err := dataStorage.GetByApplicationEUI(app.AppEUI, 10)
	if err != nil {
		t.Fatal("Error retrieving from application: ", err)
	}
	count := 0
	for data := range appChan {
		if data.Equals(firstData) {
			count++
		}
		if data.Equals(secondData) {
			count++
		}
	}
	if count != 2 {
		t.Fatal("Missing data on application channel. Expected 2 got ", count)
	}
}
