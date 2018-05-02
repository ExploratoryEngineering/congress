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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/telenordigital/goconnect"
)

func TestTokenChecker(t *testing.T) {
	server := createTestServer(noAuthConfig)
	// Set up memory storage with a token
	newToken, err := model.NewAPIToken("007", "/something", false)
	if err != nil {
		t.Fatal("Got error creating token: ", err)
	}
	if err := server.context.Storage.Token.Put(newToken, model.SystemUserID); err != nil {
		t.Fatal("Got error storing token: ", err)
	}

	testToken := func(token string, method string, path string, expectedStatus int) {
		success, message, status, _ := server.isValidToken(token, method, path)
		if status != expectedStatus {
			t.Errorf("Got %d but expected %d when requesting %s %s (msg: %s)",
				status, expectedStatus, method, path, message)
		}
		if status == http.StatusOK && !success {
			t.Errorf("Got false return with %d status code", status)
		}
	}

	// Should be OK
	testToken(newToken.Token, "GET", "/something", http.StatusOK)
	testToken(newToken.Token, "OPTIONS", "/something", http.StatusOK)
	testToken(newToken.Token, "HEAD", "/something", http.StatusOK)
	testToken(newToken.Token, "GET", "/something/else", http.StatusOK)

	// Invalid resource
	testToken(newToken.Token, "HEAD", "/other", http.StatusForbidden)
	testToken(newToken.Token, "GET", "/other/else", http.StatusForbidden)

	// Invalid token
	testToken("newToken.Token", "GET", "/something", http.StatusUnauthorized)
	testToken("newToken.Token", "GET", "/something/else", http.StatusUnauthorized)

	// No token header.
	testToken("", "GET", "/something", http.StatusUnauthorized)
	testToken("", "GET", "/something/else", http.StatusUnauthorized)

	// Illegal methods, won't get access even with the proper token
	testToken(newToken.Token, "DELETE", "/something", http.StatusForbidden)
	testToken(newToken.Token, "POST", "/something", http.StatusForbidden)
	testToken(newToken.Token, "PATCH", "/something", http.StatusForbidden)

}

var dummySession = goconnect.Session{
	UserID:        string(model.SystemUserID),
	Name:          "John Doe",
	Locale:        "en-US",
	Email:         "john@example.com",
	VerifiedEmail: true,
	Phone:         "007",
	VerifiedPhone: true,
}

// getConnectHandler injects a session into the request. Astute observers might
// notice that this method is declared on the *server* object and in a *test*
// context. Yes. It is ugly.
func (h *Server) getConnectHandler(session goconnect.Session) http.HandlerFunc {
	ret := h.handler()
	return func(w http.ResponseWriter, r *http.Request) {
		newContext := context.WithValue(r.Context(), goconnect.SessionContext, session)
		ret(w, r.WithContext(newContext))
	}
}

func TestTokenCollectionResource(t *testing.T) {
	server := createTestServer(noAuthConfig)
	server.Start()
	defer server.Shutdown()
	// Create a *new* test server that uses the connect handler from the first server
	// The token session is injected via the dummy session.
	testServer := httptest.NewServer(http.HandlerFunc(server.getConnectHandler(dummySession)))
	defer testServer.Close()

	// Add a single token.
	token, _ := model.NewAPIToken(model.SystemUserID, "/", true)
	if err := server.context.Storage.Token.Put(token, model.SystemUserID); err != nil {
		t.Fatal("Got error adding token: ", err)
	}

	rootURL := testServer.URL + "/tokens"

	invalidPosts := map[string]int{
		"Some": http.StatusBadRequest, // Invalid JSON
		"":     http.StatusBadRequest, // Blank
		`{}`:   http.StatusBadRequest, // Blank token (ignored), missing resource
		`{"Token": "barbaz" }`:                                       http.StatusBadRequest, // Missing resource
		`{"Token": "foobar", "Resource": "/some"}`:                   http.StatusCreated,    // Should work
		`{"Token": "foobarbaz", "Resource": "/some", "Write": true}`: http.StatusCreated,    // Should work
	}

	invalidGets := map[string]int{
		rootURL: http.StatusOK,
	}

	invalidMethods := []string{
		"HEAD", "OPTIONS", "PATCH", "PUT", "DELETE",
	}

	genericEndpointTest(t, rootURL, invalidGets, invalidPosts, invalidMethods)
}

func TestTokenInfoResource(t *testing.T) {
	// Set up collection of tokens, launch server
	server := createTestServer(noAuthConfig)

	// Set up a server that injects the connect session object
	testServer := httptest.NewServer(http.HandlerFunc(server.getConnectHandler(dummySession)))
	defer testServer.Close()

	// Add a single token.
	token, _ := model.NewAPIToken(model.SystemUserID, "/", true)
	if err := server.context.Storage.Token.Put(token, model.SystemUserID); err != nil {
		t.Fatal("Got error adding token: ", err)
	}

	rootURL := testServer.URL + "/tokens"

	invalidPosts := map[string]int{
	// No POSTs on this endpoint
	}

	invalidGets := map[string]int{
		rootURL + "/" + token.Token: http.StatusOK,
		rootURL + "/abcd":           http.StatusNotFound,
	}

	invalidMethods := []string{
		"HEAD", "OPTIONS", "PATCH", "POST",
	}

	genericEndpointTest(t, rootURL+"/"+token.Token, invalidGets, invalidPosts, invalidMethods)

	genericPutRequest(t, rootURL+"/"+token.Token, map[string]interface{}{
		"resource": "/foof",
		"write":    true,
		"tags":     map[string]string{"name": "value"},
	}, http.StatusOK)

	// Test DELETE method here
	testDelete(t, map[string]int{
		rootURL + "/" + token.Token: http.StatusNoContent,
		rootURL + "/foofoo":         http.StatusNotFound,
	})

	testDelete(t, map[string]int{
		rootURL + "/" + token.Token: http.StatusNotFound,
	})

}
