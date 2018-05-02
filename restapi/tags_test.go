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
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ExploratoryEngineering/congress/model"
)

func tagResourceTest(t *testing.T, rootURL string) {
	// ------------------------------------------------------------------------
	// Create a few tags
	createTag := func(nameValue map[string]string, expectedStatus int) {
		buf, _ := json.Marshal(nameValue)
		newTagBody := strings.NewReader(string(buf))
		response, err := http.Post(rootURL, "application/json", newTagBody)

		if err != nil {
			t.Fatal("Got error storing tag: ", err)
		}

		if response.StatusCode != expectedStatus {
			t.Fatalf("Expected %d but got %d from POST to %s with value %s",
				expectedStatus, response.StatusCode, rootURL, string(buf))
		}
	}
	createTag(map[string]string{"Name": "Value"}, http.StatusCreated)
	createTag(map[string]string{"Hello": "World"}, http.StatusCreated)
	createTag(map[string]string{"Key": "Value"}, http.StatusCreated)
	createTag(map[string]string{"name": "Other name"}, http.StatusConflict)
	createTag(map[string]string{"naMe": "Othername"}, http.StatusConflict)
	createTag(map[string]string{"Name ": "Other"}, http.StatusConflict)

	// ------------------------------------------------------------------------
	// Retrieve the tags
	checkURL := func(name string, expected string, expectedStatus int) {
		tagURL := rootURL + "/" + name
		resp, err := http.Get(tagURL)
		if err != nil {
			t.Fatal("Got error retrieving tag: ", err)
		}
		if resp.StatusCode != expectedStatus {
			t.Fatalf("Didn't get %d when GETting tag at %s. Got %d", expectedStatus, tagURL, resp.StatusCode)
		}
		if expectedStatus != http.StatusOK {
			return
		}
		if resp.Header.Get("Content-Type") != "text/plain" {
			t.Fatalf("Expected text/plain response but didn't get it")
		}
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal("Got error reading body: ", err)
		}
		value := string(buf)
		if value != expected {
			t.Fatalf("Got %s but expected %s when querying for %s", value, expected, name)
		}
	}
	checkURL("Name", "Value", http.StatusOK)
	checkURL("name", "Value", http.StatusOK)
	checkURL("NAME", "Value", http.StatusOK)
	checkURL("Hello", "World", http.StatusOK)
	checkURL("Key", "Value", http.StatusOK)
	checkURL("Not", "", http.StatusNotFound)
	checkURL("Found", "", http.StatusNotFound)

	// ------------------------------------------------------------------------
	// Retrieve the collection
	retrieveList := func() {
		resp, err := http.Get(rootURL)
		if err != nil {
			t.Fatal("Got error retrieving tags ", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatal("Did not get 200 OK when retrieving tag collection")
		}
		body, _ := ioutil.ReadAll(resp.Body)
		tagCollection := make(map[string]string)
		json.Unmarshal(body, &tagCollection)
		checkNV := func(name string, expected string) {
			value, ok := tagCollection[strings.ToLower(name)]
			if !ok {
				t.Fatalf("Could not find a key named %s", name)
			}
			if value != expected {
				t.Fatalf("%s did not have the expected value", name)
			}
		}
		checkNV("Name", "Value")
		checkNV("Hello", "World")
		checkNV("Key", "Value")
	}

	retrieveList()

	// ------------------------------------------------------------------------
	// Ensure invalid tag names and values are rejected
	createTag(map[string]string{"invalidChars()": "Value"}, http.StatusBadRequest)
	createTag(map[string]string{"inject": "<script>alert(1);</script>"}, http.StatusBadRequest)

	// ------------------------------------------------------------------------
	// Remove the tags
	removeTag := func(name string, expectedCode int) {
		client := http.Client{}
		//emptyBody := strings.NewReader("")
		url := rootURL + "/" + name
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			t.Fatal("Could not delete item: ", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal("Got error doing request: ", err)
		}
		if resp.StatusCode != expectedCode {
			t.Fatalf("Got %d but expected %d when calling DELETE on %s", resp.StatusCode, expectedCode, url)
		}
	}

	removeTag("Name", http.StatusNoContent)
	removeTag("Hello", http.StatusNoContent)
	removeTag("keY", http.StatusNoContent)
	removeTag("key", http.StatusNotFound)
}

func TestGatewayTagsResource(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	newGateway := model.Gateway{
		GatewayEUI: makeRandomEUI(),
		IP:         net.ParseIP("127.0.0.1"),
		StrictIP:   false,
		Tags:       model.NewTags(),
	}
	if err := h.context.Storage.Gateway.Put(newGateway, model.SystemUserID); err != nil {
		t.Fatal("Could not store gateway: ", err)
	}

	rootURL := h.loopbackURL() + "/gateways/" + newGateway.GatewayEUI.String() + "/tags"
	tagResourceTest(t, rootURL)
}

func TestApplicationTagsResource(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	newApplication := model.Application{
		AppEUI: makeRandomEUI(),
		Tags:   model.NewTags(),
	}
	newApplication.SetTag("created", "right now")
	newApplication.SetTag("environment", "testing i think")

	if err := h.context.Storage.Application.Put(newApplication, model.SystemUserID); err != nil {
		t.Fatal("Could not store gateway: ", err)
	}

	rootURL := h.loopbackURL() + "/applications/" + newApplication.AppEUI.String() + "/tags"
	tagResourceTest(t, rootURL)
}

func TestTokenTagsResource(t *testing.T) {
	// Since the token resource requires a session we have to create a server with a dummy session

	server := createTestServer(noAuthConfig)
	server.Start()
	defer server.Shutdown()
	// Create a *new* test server that uses the connect handler from the first server
	// The token session is injected via the dummy session.
	testServer := httptest.NewServer(http.HandlerFunc(server.getConnectHandler(dummySession)))
	defer testServer.Close()

	// Add a single token.
	token, _ := model.NewAPIToken("001", "/", true)
	if err := server.context.Storage.Token.Put(token, model.SystemUserID); err != nil {
		t.Fatal("Got error adding token: ", err)
	}

	rootURL := testServer.URL + "/tokens/" + token.Token + "/tags"
	tagResourceTest(t, rootURL)
}
