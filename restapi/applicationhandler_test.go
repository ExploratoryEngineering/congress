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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
)

func storeApplication(t *testing.T, application apiApplication, url string, expectedStatus int) apiApplication {
	bytes, err := json.Marshal(application)
	if err != nil {
		t.Fatalf("Got error marshaling application: %v", err)
	}
	reader := strings.NewReader(string(bytes))

	resp, err := http.Post(url, "application/json", reader)
	if err != nil {
		t.Fatalf("Could not POST application to %s: %v", url, err)
	}
	if resp.StatusCode != expectedStatus {
		t.Fatalf("POSTed successfully to %s but got %d (expected %d)", url, resp.StatusCode, expectedStatus)
	}

	ret := apiApplication{}
	if expectedStatus == http.StatusCreated {
		buffer, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Couldn't read response body from %s: %v", url, err)
		}
		if err := json.Unmarshal(buffer, &ret); err != nil {
			t.Fatalf("Couldn't unmarshal application: %v", err)
		}
	}

	return ret
}

// Ensure both /network/eui/application and /network/eui/application/eui works
func TestApplicationRoutes(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	// Create a new application
	app := storeApplication(t,
		apiApplication{Tags: map[string]string{"name": "Value"}},
		h.loopbackURL()+"/applications",
		http.StatusCreated)

	// Storage should contain application
	eui, err := protocol.EUIFromString(app.ApplicationEUI)
	if err != nil {
		t.Fatal("Got invalid EUI from created app: ", err)
	}
	_, err = h.context.Storage.Application.GetByEUI(eui, model.SystemUserID)
	if err != nil {
		t.Fatal("Could not locate app created through rest api: ", err)
	}

	// Retrieve the same application through its endpoint. Should match.
	url := h.loopbackURL() + "/applications/" + app.ApplicationEUI
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Could not GET application from %s: %v", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expeced 200 OK from %s but got %d", url, resp.StatusCode)
	}

	copy := apiApplication{}
	buffer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Couldn't read response body from %s: %v", url, err)
	}
	if err := json.Unmarshal(buffer, &copy); err != nil {
		t.Fatalf("Couldn't unmarshal application: %v", err)
	}

	if !copy.equals(app) {
		t.Fatalf("Didn't get the same app from the endpoint as the one that was created %v != %v", copy, app)
	}

	otherApp := storeApplication(t,
		apiApplication{Tags: make(map[string]string)},
		h.loopbackURL()+"/applications",
		http.StatusCreated)

	// Retrieve list, check that list contains both apps
	url = h.loopbackURL() + "/applications"
	if resp, err = http.Get(url); err != nil {
		t.Fatalf("Could not GET application list from %s: %v", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Did not get 200 OK from GET. Got %d", resp.StatusCode)
	}

	list := applicationList{}
	buffer, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Could not read body of application list: ", err)
	}

	if err := json.Unmarshal(buffer, &list); err != nil {
		t.Fatal("Could not unmarshal list: ", err)
	}

	var foundOne, foundTwo bool
	for _, v := range list.Applications {
		if v.equals(app) {
			foundOne = true
		}
		if v.equals(otherApp) {
			foundTwo = true
		}
	}

	if !foundOne || !foundTwo {
		t.Fatal("List doesn't contain both applications")
	}

	// Create ten applications in the storage layer
	tmpApp := model.NewApplication()
	tmpApp.AppEUI, _ = h.context.KeyGenerator.NewAppEUI()
	h.context.Storage.Application.Put(tmpApp, model.SystemUserID)

	// Creating a new application with the EUI set should fail
	storeApplication(t, apiApplication{ApplicationEUI: tmpApp.AppEUI.String()},
		h.loopbackURL()+"/applications", http.StatusConflict)

	// Create 10 applications that will conflict with future applications
	first := tmpApp.AppEUI.ToUint64() + 1
	for i := 0; i < 15; i++ {
		a := apiApplication{ApplicationEUI: protocol.EUIFromUint64(first).String()}
		storeApplication(t, a, h.loopbackURL()+"/applications", http.StatusCreated)
		first++
	}

	// ..and if we try to create a new it would fail the first time (since the check stops at 10) then succeed
	storeApplication(t, apiApplication{}, h.loopbackURL()+"/applications", http.StatusInternalServerError)
	storeApplication(t, apiApplication{}, h.loopbackURL()+"/applications", http.StatusCreated)
}

func TestApplicationListEndpoint(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	rootURL := h.loopbackURL() + "/applications"

	// Note that the network EUI attribute is ignored since it is a part of the path
	invalidPosts := map[string]int{
		`{}`:                                  http.StatusCreated,
		`{Name:""}`:                           http.StatusBadRequest,
		`{"ApplicationEUI":"01-02xx"}`:        http.StatusBadRequest,
		`{"ApplicationEUI": "01abcdefghijk"}`: http.StatusBadRequest,
		`{"ApplicationEUI": ""}`:              http.StatusCreated,
	}

	invalidGets := map[string]int{
		h.loopbackURL() + "/nothing": http.StatusNotFound,
	}

	invalidMethods := []string{
		"HEAD",
		"PATCH",
		"PUT",
		"DELETE",
	}

	genericEndpointTest(t, rootURL, invalidGets, invalidPosts, invalidMethods)
}

func TestApplicationInfoEndpoint(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	rootURL := h.loopbackURL() + "/applications"
	application := storeApplication(t, apiApplication{}, rootURL, http.StatusCreated)

	invalidPosts := map[string]int{
	// No POST on this endpoint
	}

	invalidGets := map[string]int{
		// All parameters are ignored and no parameters in path
		rootURL + "/foo":                      http.StatusBadRequest,
		rootURL + "/00-00-00-00-11-11-11-11":  http.StatusNotFound,
		h.loopbackURL() + "/applications/foo": http.StatusBadRequest,
	}

	invalidMethods := []string{
		"HEAD",
		"PATCH",
		"POST",
	}

	genericEndpointTest(t, rootURL+"/"+application.ApplicationEUI, invalidGets, invalidPosts, invalidMethods)

	// Test updates
	appURL := rootURL + "/" + application.ApplicationEUI
	genericPutRequest(t, appURL, map[string]interface{}{
		"randomValue": "xxx",
	}, http.StatusOK)
	genericPutRequest(t, appURL, map[string]interface{}{}, http.StatusOK)
	genericPutRequest(t, appURL, map[string]interface{}{
		"tags": map[string]string{"name": "value"},
	}, http.StatusOK)
	genericPutRequest(t, appURL, map[string]interface{}{
		"tags": map[string]string{"name": "alert('Hello');"},
	}, http.StatusBadRequest)
	genericPutRequest(t, appURL, map[string]interface{}{
		"tags": map[string]interface{}{"name": true, "value": 12},
	}, http.StatusBadRequest)
	testDelete(t, map[string]int{
		rootURL + "/" + application.ApplicationEUI: http.StatusNoContent,
		rootURL + "/11-22-33-44-55-66-77-88":       http.StatusNotFound,
	})

}

func TestApplicationDataEndpoint(t *testing.T) {
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

	invalidPosts := map[string]int{
	// No POST on this endpoint
	}

	invalidAppURL := h.loopbackURL() + "/applications/00-01-02-03-04-05-06-07"

	invalidGets := map[string]int{
		appURL + "/data":                             http.StatusOK,
		invalidAppURL + "/data":                      http.StatusNotFound,
		h.loopbackURL() + "/applications/01-02/data": http.StatusBadRequest,
	}
	invalidMethods := []string{
		"HEAD",
		"PATCH",
		"PUT",
		"DELETE",
		"POST",
	}
	genericEndpointTest(t, appURL+"/data", invalidGets, invalidPosts, invalidMethods)
}
