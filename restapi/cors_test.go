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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORS(t *testing.T) {

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if allowedOrigin == "" {
			t.Fatal("Did not get any Acess-Control-Allow-Origin header")
		}
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Fatal("Did not get any Acess-Control-Allow-Methods header")
		}
		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Fatal("Access-Control-Allow-Credentials should be set to 'true'")
		}
		if r.Header.Get("Access-Control-Request-Headers") != "" {
			if w.Header().Get("Access-Control-Allow-Headers") == "" {
				t.Fatal("Did not get any Acess-Control-Allow-Headers header")
			}
		}
		requestOrigin := r.Header.Get("Origin")
		if requestOrigin != "" {
			if allowedOrigin != requestOrigin {
				t.Fatal("Response Access-Control-Allow-Origin isn't the same as the Origin header")
			}
		}
	}
	f := addCORSHeaders(testHandler)
	{
		req := httptest.NewRequest("GET", "/", strings.NewReader(""))
		req.Header.Set("Access-Control-Request-Headers", "Range,Content-Type,X-Show-Me-Your-Internals")
		w := httptest.NewRecorder()
		f(w, req)
	}
	{
		req := httptest.NewRequest("GET", "/", strings.NewReader(""))
		req.Header.Set("Origin", "http://totally.nonsuspect.site.example.com/")
		w := httptest.NewRecorder()
		f(w, req)
	}
	{
		req := httptest.NewRequest("OPTIONS", "/", strings.NewReader(""))
		req.Header.Set("Origin", "http://totally.nonsuspect.site.example.com/")
		w := httptest.NewRecorder()
		f(w, req)
	}

}
