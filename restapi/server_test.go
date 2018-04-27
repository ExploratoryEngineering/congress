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
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage/memstore"
	"github.com/ExploratoryEngineering/pubsub"
)

var ma = protocol.MA{Prefix: [5]byte{0, 1, 3, 4, 5}, Size: protocol.MALarge}

// Configuration without authentication
var noAuthConfig = server.Configuration{HTTPServerPort: 0, DisableAuth: true, MemoryDB: true}

func createTestServer(config server.Configuration) *Server {
	ma, _ := protocol.NewMA([]byte{1, 2, 3})

	// NetID
	netID := uint32(0)

	store := memstore.CreateMemoryStorage(0, 0)

	keygen, _ := server.NewEUIKeyGenerator(ma, netID, store.Sequence)

	fob := server.NewFrameOutputBuffer()

	appRouter := pubsub.NewEventRouter(5)
	context := &server.Context{
		Storage:      &store,
		FrameOutput:  &fob,
		KeyGenerator: &keygen,
		AppRouter:    &appRouter,
		AppOutput:    server.NewAppOutputManager(&appRouter),
		Config:       &config,
	}

	server, _ := NewServer(true, context, &config)
	return server
}

func TestServerStartupNoAuth(t *testing.T) {
	h := createTestServer(noAuthConfig)
	h.Start()
	defer h.Shutdown()

	// Request the root URL
	if res, err := http.Get(h.loopbackURL()); err != nil {
		t.Errorf("Error GETting root resource: %v", err)
	} else {
		if res.StatusCode != http.StatusOK {
			t.Errorf("Got status %d. Expected 200 OK", res.StatusCode)
		}
		if _, err := ioutil.ReadAll(res.Body); err != nil {
			t.Fatal("Could not read response body: ", err)
		}
		// Content isn't that important
	}

	// Request the status URL
	if res, err := http.Get(h.loopbackURL() + "/status"); err != nil {
		t.Errorf("Error GETting status resource: %v", err)
	} else {
		if res.StatusCode != http.StatusOK {
			t.Errorf("Got status %d. Expected 200 OK", res.StatusCode)
		}
		if _, err := ioutil.ReadAll(res.Body); err != nil {
			t.Fatal("Could not read the response body: ", err)
		}
	}
}

// Enable authentication and ensure it starts as expected (and authenticates)
func TestServerStartupWithAuth(t *testing.T) {
	h := createTestServer(server.Configuration{HTTPServerPort: 0})
	h.Start()
	defer h.Shutdown()

	res, err := http.Get(h.loopbackURL())
	if err != nil {
		t.Fatalf("Couldn't retrieve root resource: %v", err)
	}

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected %d response for / but got %d", http.StatusUnauthorized, res.StatusCode)
	}

	res, err = http.Get(h.loopbackURL() + "/applications")
	if err != nil {
		t.Fatalf("Couldn't retrieve application resource: %v", err)
	}

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected %d response for /applications but got %d", http.StatusUnauthorized, res.StatusCode)
	}

	// Inject application token into application request
	token, _ := model.NewAPIToken(model.SystemUserID, "/", true)
	h.context.Storage.Token.Put(token, model.SystemUserID)

	body := strings.NewReader("")
	req, _ := http.NewRequest(http.MethodGet, h.loopbackURL()+"/applications", body)
	req.Header.Add("X-API-Token", token.Token)

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Couldn't do GET request")
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Didn't get %d with token but got %d", http.StatusOK, res.StatusCode)
	}
}
func makeRandomEUI() protocol.EUI {
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	ret := protocol.EUI{}
	copy(ret.Octets[:], randomBytes)
	return ret
}

func makeRandomKey() protocol.AESKey {
	var keyBytes [16]byte
	rand.Read(keyBytes[:])
	return protocol.AESKey{Key: keyBytes}
}

func checkContentType(t *testing.T, url string, resp *http.Response) {
	if resp.StatusCode >= 200 && resp.StatusCode <= 300 {
		if resp.Header.Get("Content-Type") != "application/json" {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Fatalf("Expected content-header for %s to say application/json but it says %s with body %s", url, resp.Header.Get("Content-Type"), string(body))
		}
	}
}

// This is a generic endpoint test. It will test invalid URLs (if the URLs
// contains path parameters), invalid POSTs (to the root URL) and verify that
// invalid methods return the appropriate status code (405)
func genericEndpointTest(t *testing.T, rootURL string, invalidGets map[string]int,
	invalidPosts map[string]int, invalidMethods []string) {

	for url, expectedValue := range invalidGets {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Got error %v when GETting %s", err, url)
		}
		if resp.StatusCode != expectedValue {
			body, err := ioutil.ReadAll(resp.Body)
			var output = "<no output>"
			if err == nil {
				output = string(body)
			}
			t.Fatalf("Got status code %d but expected %d for URL %s (body is %s)",
				resp.StatusCode, expectedValue, url, strings.TrimSpace(output))
		}

		checkContentType(t, url, resp)
	}

	// Test POST against the root URL
	for body, expectedValue := range invalidPosts {
		reader := strings.NewReader(body)
		resp, err := http.Post(rootURL, "application/json", reader)
		if err != nil {
			t.Fatalf("Error retrieving URL %s: %v", rootURL, err)
		}
		if resp.StatusCode != expectedValue {
			buf, _ := ioutil.ReadAll(resp.Body)
			t.Errorf("Got %d but expected %d when POSTing %s (response body is %s)", resp.StatusCode, expectedValue, body, string(buf))
		}
		checkContentType(t, rootURL, resp)
	}

	for _, method := range invalidMethods {
		emptyBody := strings.NewReader("")
		req, err := http.NewRequest(method, rootURL, emptyBody)
		if err != nil {
			t.Fatal("Got error creating request: ", err)
		}
		client := &http.Client{}
		req.Header.Add("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal("Got error performing request: ", err)
		}

		if resp.StatusCode != http.StatusMethodNotAllowed {
			buf := make([]byte, 100)
			resp.Body.Read(buf)
			t.Fatalf("Didn't get 405 Method Not Allowed when doing %s (got %d with body '%s')", method, resp.StatusCode, string(buf))
		}
	}
}

func testDelete(t *testing.T, testData map[string]int) {
	for url, expected := range testData {
		client := http.Client{}
		emptyBody := strings.NewReader("")
		req, err := http.NewRequest("DELETE", url, emptyBody)
		if err != nil {
			t.Fatal("Got error creating request: ", err)
		}
		req.Header.Add("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Got error calling DELETE on resource %s: %v", url, err)
		}
		if resp.StatusCode != expected {
			t.Fatalf("Expected %d but got %d when calling DELETE on %s", expected, resp.StatusCode, url)
		}
	}
}

// A generic put method for resources; exercises the readers
func genericPutRequest(t *testing.T, rootURL string, values map[string]interface{}, expected int) {
	buf, _ := json.Marshal(values)
	client := &http.Client{}
	body := bytes.NewReader(buf)
	req, err := http.NewRequest("PUT", rootURL, body)
	if err != nil {
		t.Fatal("Got error creating request: ", err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Got error performing PUT request ", err)
	}
	if resp.StatusCode != expected {
		buf, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("Didn't get the expected result. Expected %d but got %d (body = %s)", expected, resp.StatusCode, string(buf))
	}
}
