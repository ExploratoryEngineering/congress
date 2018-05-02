package main

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
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/server"
)

var (
	memoryConfig  = &server.Configuration{DisableAuth: true, MA: "00-00-00", MemoryDB: true, OnlyLoopback: true, LogLevel: 3}
	syslogConfig  = &server.Configuration{DisableAuth: true, MA: "00-00-00", MemoryDB: true, OnlyLoopback: true, LogLevel: 3, Syslog: true}
	invalidConfig = &server.Configuration{MemoryDB: false, DBConnectionString: "", OnlyLoopback: true, LogLevel: 9}
)

func testWithConfig(t *testing.T, config *server.Configuration) {
	s, err := NewServer(config)
	if err != nil {
		t.Fatalf("Got error creating congress server: %v", err)
	}

	if err := s.Start(); err != nil {
		t.Fatalf("Got error starting congress server: %v", err)
	}

	<-time.After(500 * time.Millisecond)
	if err := s.Shutdown(); err != nil {
		t.Fatalf("Got error shutting down Congress server: %v", err)
	}
}

func TestCongressServer(t *testing.T) {
	testWithConfig(t, memoryConfig)
	testWithConfig(t, syslogConfig)
}

func TestCongressServerBadConfig(t *testing.T) {
	_, err := NewServer(invalidConfig)
	if err == nil {
		t.Fatalf("Expected error with bad config but didn't get it")
	}
}
