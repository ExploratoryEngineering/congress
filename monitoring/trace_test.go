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
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Test the body read bits separately
func TestDurationRead(t *testing.T) {
	if getDuration(strings.NewReader("1")) != 1 {
		t.Fatal("Not 1")
	}
	if getDuration(strings.NewReader("")) != 0 {
		t.Fatal("Not 0")
	}
	if getDuration(strings.NewReader("a long time")) != 0 {
		t.Fatal("Not 0")
	}
	if getDuration(strings.NewReader("60")) != 60 {
		t.Fatal("Not 60")
	}

}
func TestTracing(t *testing.T) {
	ep, err := NewEndpoint(true, 0, false, true)
	if err != nil {
		t.Fatalf("Got error creating endpoint: %v", err)
	}

	if err := ep.Start(); err != nil {
		t.Fatalf("Got error starting endpoint: %v", err)
	}

	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/debug/trace", ep.Port()), "text/plain", strings.NewReader("1"))
	if err != nil {
		t.Fatalf("Couldn't do request to trace: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Got %d from trace resource. Expected %d", resp.StatusCode, http.StatusOK)
	}

	// A post right after will yield a 409
	resp, err = http.Post(fmt.Sprintf("http://localhost:%d/debug/trace", ep.Port()), "text/plain", strings.NewReader("1"))
	if err != nil {
		t.Fatalf("Couldn't do request to trace: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("Got %d from trace resource. Expected %d", resp.StatusCode, http.StatusConflict)
	}

	resp, err = http.Post(fmt.Sprintf("http://localhost:%d/debug/trace", ep.Port()), "text/plain", strings.NewReader("a long long time"))
	if err != nil {
		t.Fatalf("Couldn't do request to trace: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Got %d from trace resource. Expected %d", resp.StatusCode, http.StatusBadRequest)
	}

	time.Sleep(1 * time.Second)

	// Remove old trace files
	f, _ := os.Open(".")
	list, _ := f.Readdir(99)
	for _, v := range list {
		if strings.HasPrefix(v.Name(), "trace_congress_20") {
			t.Logf("Removing %s\n", v.Name())
			os.Remove(v.Name())
		}
	}
}
