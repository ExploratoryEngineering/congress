package server

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
	"encoding/hex"
	"strings"
	"testing"

	"github.com/ExploratoryEngineering/congress/protocol"
)

var defaultMA protocol.MA

func init() {
	tmp := strings.Replace("00-09-09", "-", "", -1)
	tmp2, _ := hex.DecodeString(tmp)
	defaultMA, _ = protocol.NewMA(tmp2)
}

// Test custom parsing for all parameters. These cannot be tested multiple
// times or with custom parameters but I'm going to assume it works.
func TestCommandLineConfigDefaults(t *testing.T) {
	config := NewDefaultConfig()
	config.MemoryDB = true
	if err := config.Validate(); err != nil {
		t.Fatalf("Expected config to be valid: %v", err)
	}
}

func TestValidConfiguration(t *testing.T) {
	config := NewMemoryNoAuthConfig()
	config.MA = "foof"
	if err := config.Validate(); err == nil {
		t.Fatal("Expected error from invalid MA string")
	}
	config.MA = "00-09-09-09-09-09-09"
	if err := config.Validate(); err == nil {
		t.Fatal("Expected error from too long MA string")
	}

	config.MA = DefaultMA
	config.TLSCertFile = "foof.cert"
	if err := config.Validate(); err == nil {
		t.Fatal("Expected error when TLS key file is missing")
	}
	config.TLSCertFile = ""
	config.DisableAuth = true
	if err := config.Validate(); err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	config.TLSCertFile = "cert"
	config.TLSKeyFile = "key"
	config.DisableAuth = false
	if err := config.Validate(); err != nil {
		t.Fatalf("Shouldn't get an error with TLS cert file and key file set")
	}

	config.DBConnectionString = ""
	config.MemoryDB = false
	if err := config.Validate(); err == nil {
		t.Fatalf("Expected error with no backend selected")
	}

}

func TestMAInvalidString(t *testing.T) {
	config := NewDefaultConfig()
	config.MA = DefaultMA
	config.RootMA()

	config.MA = "foof"
	defer func() {
		if n := recover(); n == nil {
			t.Fatal("Should have gotten a panic")
		}
	}()

	config.RootMA() // should panic
	t.Fatal("I expected panic here")

}

func TestMAInvalidMA(t *testing.T) {
	config := NewDefaultConfig()
	config.MA = "01-02-03-04-05-06-07-08-09"
	defer func() {
		if n := recover(); n == nil {
			t.Fatal("Should have gotten a panic")
		}
	}()

	config.RootMA() // should panic
	t.Fatal("I expected panic here")
}

func TestInvalidMemoryLatency(t *testing.T) {
	config := NewDefaultConfig()
	config.MemoryDB = true
	config.MemoryMinLatencyMs = 100
	config.MemoryMaxLatencyMs = 50
	if config.Validate() == nil {
		t.Fatal("max > min; no error")
	}
	config.MemoryMaxLatencyMs = 100
	if config.Validate() == nil {
		t.Fatal("max == min && max > 0; no error")
	}
	config.MemoryMaxLatencyMs = 150
	config.MemoryMinLatencyMs = 50
	if err := config.Validate(); err != nil {
		t.Fatalf("min/max valid but it fails: %v", err)
	}

	config.MemoryMaxLatencyMs = 0
	config.MemoryMinLatencyMs = 0
	if err := config.Validate(); err != nil {
		t.Fatalf("min/max 0 but it fails: %v", err)
	}

}

func TestInvalidACMEConfig(t *testing.T) {
	config := NewDefaultConfig()
	config.MemoryDB = true
	config.ACMECert = true
	if err := config.Validate(); err == nil {
		t.Fatal("Expected error when no ACME host name is set")
	}
	config.ACMEHost = "host.example.com"
	if err := config.Validate(); err != nil {
		t.Fatal("Did not expect error when ACME host name is set: ", err)
	}
}
