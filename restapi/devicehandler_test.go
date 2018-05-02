package restapi

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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
)

func storeDevice(t *testing.T, device apiDevice, url string, expectedStatus int) apiDevice {
	bytes, err := json.Marshal(device)
	if err != nil {
		t.Fatalf("Got error marshalling device: %v", err)
	}

	reader := strings.NewReader(string(bytes))

	resp, err := http.Post(url, "application/json", reader)
	if err != nil {
		t.Fatalf("Could not POST device to %s: %v", url, err)
	}
	if resp.StatusCode != expectedStatus {
		t.Fatalf("POST successfully to %s but response code was %d, not %d", url, resp.StatusCode, expectedStatus)
	}

	ret := apiDevice{}
	if expectedStatus == http.StatusCreated {
		buffer, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Couldn't read response body from %s: %v", url, err)
		}
		if err := json.Unmarshal(buffer, &ret); err != nil {
			t.Fatalf("Couldn't unmarshal device from %s: %v", url, err)
		}
		devaddr, err := protocol.DevAddrFromString(ret.DevAddr)
		if err != nil {
			t.Fatal("Error decoding devaddr: ", err)
		}
		if devaddr.ToUint32() == 0 {
			t.Fatal("DevAddr == 0. LMiC hates this")
		}
	}
	return ret
}

// Compare public fields on both, no more
func compareDevices(d1 apiDevice, d2 apiDevice) bool {
	if d1.AppSKey != d2.AppSKey ||
		d1.DevAddr != d2.DevAddr ||
		d1.DeviceEUI != d2.DeviceEUI ||
		d1.DeviceType != d2.DeviceType ||
		d1.FCntDn != d2.FCntDn ||
		d1.FCntUp != d2.FCntUp ||
		d1.NwkSKey != d2.NwkSKey ||
		d1.RelaxedCounter != d2.RelaxedCounter {
		return false
	}
	return true
}

func TestDeviceList(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	appURL := h.loopbackURL() + "/applications"
	application := storeApplication(t, apiApplication{}, appURL, http.StatusCreated)

	rootURL := appURL + "/" + application.ApplicationEUI + "/devices"

	firstDevice := storeDevice(t, apiDevice{RelaxedCounter: true}, rootURL, http.StatusCreated)
	// Ensure the device is created in the storage backend
	eui, err := protocol.EUIFromString(firstDevice.DeviceEUI)
	if err != nil {
		t.Fatal("Returned EUI is incorrect: ", err)
	}
	fd, err := h.context.Storage.Device.GetByEUI(eui)
	if err != nil {
		t.Fatalf("Could not locate the device (%s) in the backend: %v", firstDevice.eui.String(), err)
	}

	if !compareDevices(newDeviceFromModel(&fd), firstDevice) {
		t.Fatalf("Retrieved device does not match stored device %v != %v", newDeviceFromModel(&fd), firstDevice)
	}
	// Create another device
	secondDevice := storeDevice(t, apiDevice{RelaxedCounter: false}, rootURL, http.StatusCreated)

	// Get list of devices, ensure they're in the list
	resp, err := http.Get(rootURL)
	if err != nil {
		t.Fatal("Got error GETting list of devices: ", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Didn't get 200 OK from list endpoint. Got %d", resp.StatusCode)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Got error reading body from GET request: ", err)
	}
	list := deviceList{}
	if json.Unmarshal(buf, &list) != nil {
		t.Fatal("Got error unmarshaling body")
	}

	var foundOne, foundTwo bool
	for _, d := range list.Devices {
		if compareDevices(d, firstDevice) {
			foundOne = true
		}
		if compareDevices(d, secondDevice) {
			foundTwo = true
		}
	}
	if !foundOne || !foundTwo {
		t.Fatal("Didn't find the devices")
	}

	// Get the device from the info URL
	deviceURL := rootURL + "/" + secondDevice.DeviceEUI
	resp, err = http.Get(deviceURL)
	if err != nil {
		t.Fatalf("Got error reading device from info URL (%s): %v", deviceURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Didn't get 200 OK from device info URL (%s). Got %d", deviceURL, resp.StatusCode)
	}

	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Error reading body from GET: ", err)
	}

	retrievedDevice := apiDevice{}
	if json.Unmarshal(buf, &retrievedDevice) != nil {
		t.Fatal("Error unmarshaling device")
	}

	if !compareDevices(retrievedDevice, secondDevice) {
		t.Fatalf("Retrieved device didn't match stored device: %v != %v", retrievedDevice, secondDevice)
	}

	// ...finally look for a device that doesn't exist
	deviceURL = rootURL + "/00-00-01-01-02-02-00-00"
	resp, _ = http.Get(deviceURL)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Didn't get 404 NOT FOUND. Got %d", resp.StatusCode)
	}

	tmpDevice := model.NewDevice()
	tmpDevice.DeviceEUI, _ = h.context.KeyGenerator.NewDeviceEUI()
	if err := h.context.Storage.Device.Put(tmpDevice, application.eui); err != nil {
		t.Fatal(err)
	}
	first := tmpDevice.DeviceEUI.ToUint64() + 1
	// Create 15 devices with EUIs that will conflict with the next autogenerated
	// eui
	for i := 0; i < 15; i++ {
		d := apiDevice{DeviceEUI: protocol.EUIFromUint64(first).String()}
		storeDevice(t, d, rootURL, http.StatusCreated)
		first++
	}

	// create one that will fail since there's more than 10 EUIs blocking, then
	// another one that will work since there's 15-10=5 EUIs
	storeDevice(t, apiDevice{}, rootURL, http.StatusInternalServerError)
	storeDevice(t, apiDevice{}, rootURL, http.StatusCreated)
}

func TestDeviceListEndpoint(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	appURL := h.loopbackURL() + "/applications"
	application := storeApplication(t, apiApplication{}, appURL, http.StatusCreated)

	rootURL := appURL + "/" + application.ApplicationEUI + "/devices"

	invalidPosts := map[string]int{
		// Invalid JSON
		"Some": http.StatusBadRequest,
		// Blank
		"": http.StatusBadRequest,
		// Invalid EUI
		`{"DeviceEUI": "xx"}`: http.StatusBadRequest,
		// DevAddr will be rejected
		`{"DevAddr": "bar", "DeviceType": "ABP"}`: http.StatusBadRequest,
		// AppSKey is checked next
		`{"DevAddr": "01020304", "AppSKey": "foo", "DeviceType": "ABP"}`: http.StatusBadRequest,
		// Invalid network session key
		`{"DevAddr": "01020304", "AppSKey": "01020304 05060708 01020304 05060708", "NwkSKey": "bar", "DeviceType": "ABP"}`: http.StatusBadRequest,
		// This would be OK. All defaults used
		"{}": http.StatusCreated,
		// Overriding the EUI should also work
		`{"DeviceEUI": "01-02-03-04-05-06-07-aa"}`:                                http.StatusCreated,
		`{"AppSKey": "01020304 05060708 01020304 05060708", "DeviceType": "ABP"}`: http.StatusCreated,
		`{"NwkSKey": "01020304 05060708 01020304 05060708", "DeviceType": "ABP"}`: http.StatusCreated,
	}

	invalidGets := map[string]int{
		rootURL: http.StatusOK,
		h.loopbackURL() + "/applications/bar/devices/baz": http.StatusBadRequest,
		h.loopbackURL() + "/applications/bar/devices":     http.StatusBadRequest,
	}

	invalidMethods := []string{
		"HEAD",
		"PATCH",
		"PUT",
		"DELETE",
	}

	genericEndpointTest(t, rootURL, invalidGets, invalidPosts, invalidMethods)
}

func TestDeviceInfoEndpoint(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	appURL := h.loopbackURL() + "/applications"
	application := storeApplication(t, apiApplication{}, appURL, http.StatusCreated)

	deviceURL := appURL + "/" + application.ApplicationEUI + "/devices"

	device1 := storeDevice(t, apiDevice{}, deviceURL, http.StatusCreated)
	device2 := storeDevice(t, apiDevice{
		DeviceType: "ABP",
		AppSKey:    "01020304050607080102030405060708",
		NwkSKey:    "01020304050607080102030405060708",
		DevAddr:    "01020304"}, deviceURL, http.StatusCreated)

	// Add some data to the device. One is enough.
	err := h.context.Storage.DeviceData.Put(device1.eui, model.DeviceData{
		DeviceEUI:  device1.eui,
		Timestamp:  1,
		Data:       []byte{0, 1, 2, 3, 4},
		Frequency:  868.1,
		SNR:        10.2,
		RSSI:       -120,
		DataRate:   "SF7BW125",
		GatewayEUI: protocol.EUIFromUint64(0),
	})

	if err != nil {
		t.Errorf("Got error storing device data: %v", err)
	}
	rootURL := appURL + "/" + application.ApplicationEUI + "/devices/" + device1.DeviceEUI

	invalidPosts := map[string]int{
	// No POST for this endpoint
	}

	invalidGets := map[string]int{
		rootURL: http.StatusOK,
		h.loopbackURL() + "/applications/bar/devices/baz":                                http.StatusBadRequest,
		h.loopbackURL() + "/applications/" + application.ApplicationEUI + "/devices/baz": http.StatusBadRequest,
		h.loopbackURL() + "/applications/bar/devices":                                    http.StatusBadRequest,
		h.loopbackURL() + "/applications/00-00-00-00-00-00-00-00/devices":                http.StatusNotFound,
	}

	invalidMethods := []string{
		"HEAD",
		"PATCH",
		"POST",
	}

	genericEndpointTest(t, rootURL, invalidGets, invalidPosts, invalidMethods)

	// Test source output
	rootURL2 := appURL + "/" + application.ApplicationEUI + "/devices/" + device2.DeviceEUI
	resp, err := http.Get(rootURL + "/source")
	if err != nil {
		t.Fatalf("Got error retrieving device source: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Got %d retrieving device source", resp.StatusCode)
	}
	resp, err = http.Get(rootURL2 + "/source")
	if err != nil {
		t.Fatalf("Got error retrieving device source: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Got %d retrieving device source", resp.StatusCode)
	}

	genericPutRequest(t, rootURL, map[string]interface{}{
		"devAddr":        "01020304",
		"appKey":         "0000 1111 2222 3333 4444 5555 6666 aaaa",
		"nwkSKey":        "0000 1111 2222 3333 4444 5555 6666 7777",
		"appSKey":        "1111 2222 3333 4444 5555 6666 7777 8888",
		"relaxedCounter": true,
		"fCntUp":         99,
		"fCntDn":         100,
	}, http.StatusOK)
	genericPutRequest(t, rootURL, map[string]interface{}{
		"devAddr": "foo",
	}, http.StatusBadRequest)
	genericPutRequest(t, rootURL, map[string]interface{}{
		"appKey": "abc",
	}, http.StatusBadRequest)
	genericPutRequest(t, rootURL, map[string]interface{}{
		"nwkSKey": "abc",
	}, http.StatusBadRequest)
	genericPutRequest(t, rootURL, map[string]interface{}{
		"appSKey": "abc",
	}, http.StatusBadRequest)
	genericPutRequest(t, rootURL, map[string]interface{}{
		"tags": map[string]string{"name": "value"},
	}, http.StatusOK)
	genericPutRequest(t, rootURL, map[string]interface{}{
		"tags": map[string]string{"name": "alert('Hello');"},
	}, http.StatusBadRequest)
	genericPutRequest(t, rootURL, map[string]interface{}{
		"tags": map[string]interface{}{"name": true, "value": 12},
	}, http.StatusBadRequest)

	// Test delete method
	testDelete(t, map[string]int{
		deviceURL + "/" + device1.DeviceEUI:    http.StatusNoContent,
		deviceURL + "/" + device2.DeviceEUI:    http.StatusNoContent,
		deviceURL + "/00-11-22-33-44-55-66-77": http.StatusNotFound,
	})
}

func TestDeviceDataEndpoint(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	appsURL := h.loopbackURL() + "/applications"
	application := storeApplication(t, apiApplication{}, appsURL, http.StatusCreated)

	appURL := appsURL + "/" + application.ApplicationEUI
	device := storeDevice(t, apiDevice{}, appURL+"/devices", http.StatusCreated)

	eui, _ := protocol.EUIFromString(device.DeviceEUI)
	for i := 0; i < 10; i++ {
		err := h.context.Storage.DeviceData.Put(eui, model.DeviceData{
			DeviceEUI:  eui,
			Timestamp:  int64(i),
			Data:       []byte{0, 1, 2, 3, 4, 5},
			GatewayEUI: eui,
		})
		if err != nil {
			t.Fatal("Got error storing device data: ", err)
		}
	}

	deviceURL := appURL + "/devices/" + device.DeviceEUI

	invalidPosts := map[string]int{
	// No POST on this endpoint
	}

	urlTemplate := h.loopbackURL() + "/applications/%s/devices/%s/data"
	const invalidEUI string = "00-00-00-00-00-00-00-00"
	const improperEUI string = "00-00"

	validURL := fmt.Sprintf(urlTemplate, application.ApplicationEUI, device.DeviceEUI)
	invalidDeviceEUI := fmt.Sprintf(urlTemplate, application.ApplicationEUI, invalidEUI)
	incorrectDeviceEUI := fmt.Sprintf(urlTemplate, application.ApplicationEUI, improperEUI)
	invalidAppEUI := fmt.Sprintf(urlTemplate, invalidEUI, device.DeviceEUI)
	incorrectAppEUI := fmt.Sprintf(urlTemplate, improperEUI, device.DeviceEUI)
	invalidGets := map[string]int{
		validURL:           http.StatusOK,
		invalidDeviceEUI:   http.StatusNotFound,
		incorrectDeviceEUI: http.StatusBadRequest,
		invalidAppEUI:      http.StatusNotFound,
		incorrectAppEUI:    http.StatusBadRequest,
	}
	invalidMethods := []string{
		"HEAD",
		"PATCH",
		"PUT",
		"DELETE",
		"POST",
	}
	genericEndpointTest(t, deviceURL+"/data", invalidGets, invalidPosts, invalidMethods)
}

func TestDeviceMessageInput(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	application := storeApplication(t, apiApplication{}, h.loopbackURL()+"/applications", http.StatusCreated)

	appURL := h.loopbackURL() + "/applications/" + application.ApplicationEUI
	device := storeDevice(t, apiDevice{}, appURL+"/devices", http.StatusCreated)

	eui, _ := protocol.EUIFromString(device.DeviceEUI)
	for i := 0; i < 10; i++ {
		err := h.context.Storage.DeviceData.Put(eui, model.DeviceData{
			DeviceEUI:  eui,
			Timestamp:  int64(i),
			Data:       []byte{0, 1, 2, 3, 4, 5},
			GatewayEUI: eui,
		})
		if err != nil {
			t.Fatal("Got error storing device data: ", err)
		}
	}

	deviceURL := appURL + "/devices/" + device.DeviceEUI

	invalidPosts := map[string]int{
		`x`:                         http.StatusBadRequest,
		`{}`:                        http.StatusBadRequest,
		`{"port": -1}`:              http.StatusBadRequest,
		`{"port": 254}`:             http.StatusBadRequest,
		`{"port": 999}`:             http.StatusBadRequest,
		`{"port": 1, "data": "zy"}`: http.StatusBadRequest,
		`{"port": 1, "data": ""}`:   http.StatusBadRequest,
	}

	invalidGets := map[string]int{}

	invalidMethods := []string{
		"HEAD",
		"PATCH",
		"PUT",
	}

	genericEndpointTest(t, deviceURL+"/message", invalidGets, invalidPosts, invalidMethods)

	// Reset output buffer
	h.context.Storage.DeviceData.DeleteDownstream(eui)

	// Post a single message and ensure the output buffer is set.
	reader := strings.NewReader(`{"port": 1, "data": "01AA02BB03CC04DD", "ack": true}`)
	resp, _ := http.Post(deviceURL+"/message", "application/json", reader)
	if resp.StatusCode != http.StatusCreated {
		buf, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("Got status %d with body: %s posting to %s", resp.StatusCode, string(buf), deviceURL+"/message")
	}
	msg, err := h.context.Storage.DeviceData.GetDownstream(eui)
	if err != nil {
		t.Fatalf("Expected a new message to be created but got error: %v", err)
	}

	if msg.Data != "01AA02BB03CC04DD" {
		t.Fatalf("Not the expected payload: %v", msg.Data)
	}

	// Retrieve an upstream message
	newMsg := model.NewDownstreamMessage(eui, 100)
	newMsg.Data = "000102030405"
	newMsg.Ack = true

	url := h.loopbackURL() + "/applications/" + application.ApplicationEUI + "/devices/" + device.DeviceEUI + "/message"
	h.context.Storage.DeviceData.DeleteDownstream(eui)
	resp, _ = http.Get(url)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404 NOT FOUND but got %d", resp.StatusCode)
	}
	h.context.Storage.DeviceData.PutDownstream(eui, newMsg)

	resp, _ = http.Get(url)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Did not get 200 OK from downstream message but got %d", resp.StatusCode)
	}

	downMsg := apiDownstreamMessage{}
	if err := json.NewDecoder(resp.Body).Decode(&downMsg); err != nil {
		t.Fatalf("Got error decoding upstream response: %v", err)
	}
	if !downMsg.Ack || downMsg.Data != newMsg.Data || downMsg.Port != newMsg.Port {
		t.Fatalf("Got different response from what was created. Got %v, expected %v", downMsg, newMsg)
	}

	// Call DELETE on the resource. Should return 204 NO Content, even when there's no
	// downstream message
	testDelete(t, map[string]int{
		url: http.StatusNoContent,
		url: http.StatusNoContent,
		h.loopbackURL() + "/applications/" + application.ApplicationEUI + "/devices/00/message": http.StatusBadRequest,
	})
}

func TestAutomaticDownstreamMessageRemoval(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	application := storeApplication(t, apiApplication{}, h.loopbackURL()+"/applications", http.StatusCreated)
	deviceURL := fmt.Sprintf("%s/applications/%s/devices", h.loopbackURL(), application.ApplicationEUI)
	device := storeDevice(t, apiDevice{}, deviceURL, http.StatusCreated)
	messageURL := fmt.Sprintf("%s/applications/%s/devices/%s/message", h.loopbackURL(), application.ApplicationEUI, device.DeviceEUI)

	createMessage := func(msgData string, expectedStatus int) {
		resp, err := http.Post(messageURL, "application/json", strings.NewReader(msgData))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != expectedStatus {
			t.Fatalf("Expected %d but got %d response", expectedStatus, resp.StatusCode)
		}
	}

	updateSentAck := func(sent, ack int64) {
		eui, _ := protocol.EUIFromString(device.DeviceEUI)
		if err := h.context.Storage.DeviceData.UpdateDownstream(eui, sent, ack); err != nil {
			t.Fatal(err)
		}
	}
	// Schedule a new downstream message. Should succeed.
	createMessage(`{"port": 100, "data": "aabbccdd", "ack": false}`, http.StatusCreated)

	// Schedule another message. Should fail.
	createMessage(`{"port": 101, "data": "aabbccdd", "ack": false}`, http.StatusConflict)

	// Mimic sent status by updating sent field.
	updateSentAck(12, 0)

	// Schedule another message. Should succeed since the message is sent.
	createMessage(`{"port": 102, "data": "aabbccdd", "ack": true}`, http.StatusCreated)

	// Mimic sent status
	updateSentAck(13, 0)

	// Schedule another. Should fail since the message isn't acked
	createMessage(`{"port": 103, "data": "aabbccdd", "ack": false}`, http.StatusConflict)

	// Mimic ack status
	updateSentAck(14, 15)

	// Schedule another. Should succeed since the message is acked.
	createMessage(`{"port": 104, "data": "aabbccdd", "ack": false}`, http.StatusCreated)
}
