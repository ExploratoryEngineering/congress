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
	"sync"
	"time"
)

// The memory logger logs entries to a circular buffer, overwriting the old
// elements as new elements are added. There's not much trickery here; the
// log entries are stored in a simple bounded array and a counter is used
// to keep track of the first element. Empty elements are skipped when the
// list is returned.

const maxLogItems = 10

// LogEntry is a single log entry for outputs
type LogEntry struct {
	Timestamp time.Time
	Message   string
}

// NewLogEntry creates a new log entry
func NewLogEntry(message string) LogEntry {
	return LogEntry{
		Timestamp: time.Now(),
		Message:   message,
	}
}

// TimeString converts the timestamp into a time string
func (l *LogEntry) TimeString() string {
	return l.Timestamp.Format(time.RFC3339)
}

// IsValid returns true if the message is valid
func (l *LogEntry) IsValid() bool {
	return l.Message != ""
}

// MemoryLogger logs events to a circular buffer
type MemoryLogger struct {
	mutex   *sync.Mutex
	Entries []LogEntry
	index   int
}

// NewMemoryLogger creates a new memory logger
func NewMemoryLogger() MemoryLogger {
	return MemoryLogger{
		mutex:   &sync.Mutex{},
		Entries: make([]LogEntry, 10),
		index:   0,
	}
}

// Append appends a new log item to the log
func (m *MemoryLogger) Append(newEntry LogEntry) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Entries[m.index] = newEntry
	m.index = (m.index + 1) % maxLogItems
}

// Items returns the list of items
func (m *MemoryLogger) Items() []LogEntry {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	first := m.index

	var ret []LogEntry
	for i := first; i < first+maxLogItems; i++ {
		if m.Entries[i%maxLogItems].Timestamp.IsZero() {
			continue
		}
		ret = append(ret, m.Entries[i%maxLogItems])
	}
	return ret
}
