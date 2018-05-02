package monitoring

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
	"net/http"
	"testing"
)

func TestLaunchEndpoint(t *testing.T) {
	ep, err := NewEndpoint(true, 0, true, false)
	if err != nil {
		t.Fatalf("Got error creating endpoint: %v", err)
	}

	if err := ep.Start(); err != nil {
		t.Fatalf("Got error starting endpoint: %v", err)
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", ep.Port()))
	if err != nil {
		t.Fatalf("Couldn't do request to root resource: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Got %d from root resource. Expected %d", resp.StatusCode, http.StatusOK)
	}

	// Retrieve the vars endpoint
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d%s", ep.Port(), defaultEndpoint))
	if err != nil {
		t.Fatalf("Couldn't do request to expvars: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Got %d from expvars resource. Expected %d", resp.StatusCode, http.StatusOK)
	}

	// Output should contain the keys registered
	output := make(map[string]interface{})

	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		t.Fatalf("Got error decoding output: %v", err)
	}

	if err := ep.Shutdown(); err != nil {
		t.Fatalf("Got error stopping endpoint: %v", err)
	}
}
