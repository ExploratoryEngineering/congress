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
	"fmt"
	"testing"
)

func TestMemoryLogger(t *testing.T) {
	logger := NewMemoryLogger()

	if len(logger.Items()) != 0 {
		t.Fatal("Memory log should contain 0 items but had ", len(logger.Items()))
	}

	logger.Append(NewLogEntry("1"))

	t.Logf("Items = %d", logger.index)
	t.Logf("Log = %v", logger.Entries)
	if len(logger.Items()) != 1 {
		t.Fatal("Memory log should contain 1 item but had ", len(logger.Items()))
	}

	for j := 0; j < 100; j++ {
		logger.Append(NewLogEntry(fmt.Sprintf("Item %d", j)))
	}

	if len(logger.Items()) != maxLogItems {
		t.Fatalf("Memory log should contain %d items but had %d", maxLogItems, len(logger.Items()))
	}

	const lastMessage = "This is the last log entry"
	logger.Append(NewLogEntry(lastMessage))

	items := logger.Items()

	if items[maxLogItems-1].Message != lastMessage {
		t.Fatalf("Last message should contain %s but had %s (list = %v)", lastMessage, items[maxLogItems-1].Message, items)
	}
}

func TestLogEntry(t *testing.T) {
	// Just exercise the time formatting function
	item := NewLogEntry("This is an entry")
	t.Logf("Log item = %s: %s", item.TimeString(), item.Message)
}

// Benchmark logging
func BenchmarkLogging(b *testing.B) {
	logger := NewMemoryLogger()

	for i := 0; i < b.N; i++ {
		logger.Append(NewLogEntry("something"))
	}
}

// Benchmark log list retrieval
func BenchmarkRetrieval(b *testing.B) {
	logger := NewMemoryLogger()

	for i := 0; i < 10; i++ {
		logger.Append(NewLogEntry("something"))
	}

	for i := 0; i < b.N; i++ {
		logger.Items()
	}
}
