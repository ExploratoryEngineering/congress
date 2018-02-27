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
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test simple routes
func TestRouting(t *testing.T) {
	matchingRoutes := []string{
		"/foo/first",
		"/foo/second",
		"/bar/first",
		"/bar/second",
		"/baz",
		"/baz/foo/bar",
		"/baz/foo/bar?param=value&value=param",
	}
	nonMatchingRoutes := []string{
		"/foo",
		"/foo/other/some",
		"/baz/bar",
	}

	invocationCount := 0

	routeHandler := func(w http.ResponseWriter, r *http.Request) {
		invocationCount++
	}

	router := parameterRouter{}
	router.AddRoute("/foo/{arg}", routeHandler)
	router.AddRoute("/bar/{arg}", routeHandler)
	router.AddRoute("/baz", routeHandler)
	router.AddRoute("/baz/foo/bar", routeHandler)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, route := range matchingRoutes {
		handler := router.GetHandler(route)
		if handler != nil {
			handler.ServeHTTP(w, req)
		}
	}

	for _, route := range nonMatchingRoutes {
		handler := router.GetHandler(route)
		if handler != nil {
			handler.ServeHTTP(w, req)
		}
	}

	if invocationCount != len(matchingRoutes) {
		t.Errorf("Did not get the expected number of matches. Got %d expected %d.", invocationCount, len(matchingRoutes))
	}
}

// A benchmark that both adds and routes
func BenchmarkRouting(b *testing.B) {
	rand.Seed(42)
	const routeCount int = 50
	router := parameterRouter{}
	testHandler := func(w http.ResponseWriter, r *http.Request) { /* empty */ }
	for i := 0; i < routeCount; i++ {
		route := fmt.Sprintf("/some/{arg1}/%d/{arg2}", i)
		router.AddRoute(route, testHandler)
	}

	for i := 0; i < b.N; i++ {
		randomRoute := fmt.Sprintf("/some/%d/%d/%d", rand.Intn(routeCount), rand.Intn(routeCount), rand.Intn(routeCount))
		handler := router.GetHandler(randomRoute)
		if handler == nil {
			b.Error("Did not expect nil response")
		}
	}
}

// The number of routes in the router
const routeCount int = 50

// Router for the benchmark test
var brouter parameterRouter

// Routes to test - using a fixed number
var routesToTest []string

func init() {
	brouter = parameterRouter{}

	rand.Seed(42)
	testHandler := func(w http.ResponseWriter, r *http.Request) { /* empty */ }
	for i := 0; i < routeCount; i++ {
		route := fmt.Sprintf("/some/{arg1}/%d/{arg2}", i)
		brouter.AddRoute(route, testHandler)
	}
	for i := 0; i < routeCount; i++ {
		routesToTest = append(routesToTest, fmt.Sprintf("/some/%d/%d/%d", rand.Intn(routeCount), i, rand.Intn(routeCount)))
		routesToTest = append(routesToTest, "/not/matching/route")
	}
}

// Test just the routing request; set up isn't very critical
func BenchmarkJustRouting(b *testing.B) {
	for i := 0; i < b.N; i++ {
		handler := brouter.GetHandler(routesToTest[i%routeCount])
		if handler == nil {
			b.Error("Did not expect nil response")
		}
	}
}
