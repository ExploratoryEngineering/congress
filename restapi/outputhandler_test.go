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

	"reflect"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
)

// Create a new output by POSTing to the resource
func createNewOutput(t *testing.T, url string, appEUI protocol.EUI) model.AppOutput {
	// Create some outputs on that application
	conf, _ := model.NewTransportConfig(`{"type":"log"}`)
	op1 := model.AppOutput{EUI: makeRandomEUI(), AppEUI: appEUI, Configuration: conf}

	ml := server.NewMemoryLogger()
	apiOutput := newOutputFromModel(op1, &ml, "none")
	buf, _ := json.Marshal(&apiOutput)
	b := strings.NewReader(string(buf))
	resp, err := http.Post(url, "application/json", b)
	if err != nil {
		t.Fatalf("Got error POSTing to %s: %v", url, err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Got %d, expected %d", resp.StatusCode, http.StatusCreated)
	}
	buf, _ = ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(buf, &apiOutput); err != nil {
		t.Fatalf("Got error unmarshaling response: %v (response = %s)", err, string(buf))
	}
	op1.EUI, _ = protocol.EUIFromString(apiOutput.EUI)
	return op1
}

func TestOutputHandlers(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	// Create an application
	app := model.NewApplication()
	app.AppEUI = makeRandomEUI()

	h.context.Storage.Application.Put(app, model.SystemUserID)

	appURL := h.loopbackURL() + "/applications/" + app.AppEUI.String()

	op1 := createNewOutput(t, appURL+"/outputs", app.AppEUI)

	// Get list of outputs

	resp, err := http.Get(appURL + "/outputs")
	if err != nil {
		t.Fatal("Got error querying output list: ", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK but got %d", resp.StatusCode)
	}

	buf, _ := ioutil.ReadAll(resp.Body)
	var list = apiAppOutputList{List: make([]apiAppOutput, 0)}
	if err := json.Unmarshal(buf, &list); err != nil {
		t.Fatal("Got error unmarshaling output list: ", err)
	}

	if len(list.List) != 1 {
		t.Fatalf("Expected just one output but got %d", len(list.List))
	}

	if list.List[0].AppEUI != app.AppEUI.String() || list.List[0].EUI != op1.EUI.String() || !reflect.DeepEqual(list.List[0].Config, op1.Configuration) {
		t.Fatalf("Did not get the output I expected. Got %v, expected something similar to %v", list.List[0], op1)
	}

	//***********************************************************************
	// Retrieve the single output. It should be similar
	resp, err = http.Get(appURL + "/outputs/" + op1.EUI.String())
	if err != nil {
		t.Fatal("Could not get single output: ", err)
	}

	buf, _ = ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK but got %d (body is %s)", resp.StatusCode, string(buf))
	}

	var opAPI apiAppOutput
	if err := json.Unmarshal(buf, &opAPI); err != nil {
		t.Fatalf("Got error reading output: %v (output is %s)", err, string(buf))
	}

	if opAPI.AppEUI != list.List[0].AppEUI ||
		opAPI.EUI != list.List[0].EUI ||
		!reflect.DeepEqual(opAPI.Config, list.List[0].Config) {
		t.Fatalf("Didn't get the same output. Got %v expected %v", opAPI, list.List[0])
	}

	//***********************************************************************
	// Update the configuration and PUT to the resource.
	configJSON := `{"type":"log","username": "johndoe", "password": "secret", "endpoint": "mqtt.example.com", "tls": false}`
	if op1.Configuration, err = model.NewTransportConfig(configJSON); err != nil {
		t.Fatal("Got error creating MQTT config: ", err)
	}

	ml := server.NewMemoryLogger()
	updatedOp := newOutputFromModel(op1, &ml, "uknown")

	url := appURL + "/outputs/" + op1.EUI.String()
	buf, _ = json.Marshal(&updatedOp)
	reader := strings.NewReader(string(buf))
	request, err := http.NewRequest(http.MethodPut, url, reader)

	client := &http.Client{}
	request.Header.Add("Content-Type", "application/json")
	resp, err = client.Do(request)
	if err != nil {
		t.Fatal("Got error performing PUT: ", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK when PUTing item but got %d @ %s", resp.StatusCode, url)
	}

}
