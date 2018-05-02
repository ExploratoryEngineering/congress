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
	"testing"
	"time"
)

type errorCounter struct {
	count int
	mod   int
}

func (e *errorCounter) IsError() bool {
	e.count++
	if (e.count % e.mod) == 0 {
		return false
	}
	return true
}

// Just to make sure the error counter works
func TestInternal(t *testing.T) {
	e1 := errorCounter{0, 1}
	for i := 0; i < 10; i++ {
		if e1.IsError() {
			t.Fatal("No error expected here")
		}
	}
	e2 := errorCounter{0, 2}
	for i := 0; i < 10; i++ {
		if !e2.IsError() {
			t.Fatal("Expected error")
		}
		if e2.IsError() {
			t.Fatal("No error expected here")
		}
	}
	e3 := errorCounter{0, 3}
	for i := 0; i < 10; i++ {
		if !e3.IsError() {
			t.Fatal("Expected error")
		}
		if !e3.IsError() {
			t.Fatal("Expected error")
		}
		if e3.IsError() {
			t.Fatal("No error expected here")
		}

	}
}

// This is an test implementation of a transport passes on the messages to a
// buffered channel
type testTransport struct {
	t           *testing.T
	openCount   int
	closeCount  int
	openError   errorCounter
	sendError   errorCounter
	messageChan chan interface{}
	wg          *sync.WaitGroup
}

func (l *testTransport) close(ml *MemoryLogger) {
	if ml == nil {
		l.t.Fatal("Memory logger is nil")
	}
	ml.Append(NewLogEntry("test_close"))
	l.closeCount++
	l.wg.Done()
}

func (l *testTransport) waitForClose() {
	l.wg.Wait()
}

func (l *testTransport) open(ml *MemoryLogger) bool {
	if ml == nil {
		l.t.Fatal("Memory logger is nil")
	}
	if l.openError.IsError() {
		return false
	}
	ml.Append(NewLogEntry("test_open"))
	l.t.Log("openConnection called")
	l.openCount++
	l.wg.Add(1)
	return true
}

func (l *testTransport) send(msg interface{}, ml *MemoryLogger) bool {
	if ml == nil {
		l.t.Fatal("Memory logger is nil")
	}
	if l.sendError.IsError() {
		return false
	}
	l.messageChan <- msg
	ml.Append(NewLogEntry("test_message"))
	return true
}

// Test the message dispatcher - all happy path tests
func TestMessageDispatcherHappyPath(t *testing.T) {
	o := makeRandomOutput()
	msgChannel := make(chan interface{})
	d := testTransport{t, 0, 0, errorCounter{0, 1}, errorCounter{0, 1}, make(chan interface{}, 10), &sync.WaitGroup{}}
	ml := NewMemoryLogger()
	w := newMessageDispatcher(&o, &ml, msgChannel, &d)

	w.start()

	msgChannel <- "This is the first"
	// The state should be "active" when a message is sent
	if w.status() != string(dispatcherActive) {
		t.Fatalf("State isn't %v but %v before start", dispatcherActive, w.status())
	}
	msgChannel <- "This is the second"
	msgChannel <- "This is the third"

	for i := 0; i < 3; i++ {
		select {
		case <-d.messageChan:
		// OK
		case <-time.After(100 * time.Millisecond):
			// whups. Timeout!
			t.Fatal("Got timeout waiting for messages!")
		}
	}

	w.stop()

	d.waitForClose()
	if d.openCount != 1 {
		t.Fatalf("Expected 1 open, got %d open calls", d.openCount)
	}
	if d.closeCount != 1 {
		t.Fatalf("Expected 1 close, got %d close calls", d.closeCount)
	}

	// Check the log to see if all logged messages are present
	found := 0
	for _, v := range w.logs().Items() {
		if !v.IsValid() {
			continue
		}
		if v.Message == "test_open" {
			found++
		}
		if v.Message == "test_close" {
			found++
		}
		if v.Message == "test_message" {
			found++
		}
	}
	if found != 5 {
		t.Fatalf("Found %d of 5 log messages", found)
	}

	// ..and the accessors show the right references
	if w.messageChannel() != msgChannel {
		t.Fatal("Message channel doesn't match")
	}

	if w.output() != &o {
		t.Fatal("Output has changed")
	}
}

// Do a test where the transport fails with intermittent errors
func TestDeliveryWithErrors(t *testing.T) {
	o := makeRandomOutput()
	msgChannel := make(chan interface{})
	d := testTransport{t, 0, 0, errorCounter{0, 3}, errorCounter{0, 3}, make(chan interface{}, 10), &sync.WaitGroup{}}
	ml := NewMemoryLogger()
	w := newMessageDispatcher(&o, &ml, msgChannel, &d)
	// Speed up for the test
	w.connectRetryMaxWaitTime = time.Millisecond * 1
	w.sendRetryTimeMs = 1
	w.connectRetryTime = time.Millisecond * 1
	w.start()
	msgChannel <- "First message"
	msgChannel <- "Second message"

	for i := 0; i < 2; i++ {
		select {
		case <-d.messageChan:
		// OK
		case <-time.After(5 * time.Second):
			t.Fatal("Got timeout waiting for message")
		}

	}
	w.stop()
	d.waitForClose()
}

// Ensure the dispatcher idles when there's no messages for a given period of time
func TestIdleDispatcher(t *testing.T) {
	o := makeRandomOutput()
	msgChannel := make(chan interface{})
	d := testTransport{t, 0, 0, errorCounter{0, 1}, errorCounter{0, 1}, make(chan interface{}, 10), &sync.WaitGroup{}}
	ml := NewMemoryLogger()
	w := newMessageDispatcher(&o, &ml, msgChannel, &d)
	// Speed up for the test
	w.idleTime = time.Millisecond * 100
	w.start()
	msgChannel <- "First message"

	select {
	case <-d.messageChan:
	// OK
	case <-time.After(5 * time.Second):
		t.Fatal("Got timeout waiting for message")
	}

	// Close should be called automatically when it idles long enough
	d.waitForClose()
	w.stop()
}

// Let the dispatcher fail a send
func TestFailSending(t *testing.T) {
	o := makeRandomOutput()
	msgChannel := make(chan interface{})
	d := testTransport{t, 0, 0, errorCounter{0, 1}, errorCounter{0, maxRetries * 2}, make(chan interface{}, 10), &sync.WaitGroup{}}
	ml := NewMemoryLogger()
	w := newMessageDispatcher(&o, &ml, msgChannel, &d)
	// Speed up for the test
	w.sendRetryTimeMs = 1

	w.start()
	msgChannel <- "First message"

	// Close should be called automatically when it idles long enough
	<-time.After(50 * time.Millisecond)
	w.stop()
	d.waitForClose()
	select {
	case <-d.messageChan:
		t.Fatal("Should not receive a message")
	// OK
	case <-time.After(50 * time.Millisecond):
	}

}

// Make sure the output can terminate even when it is connecting
func TestFailOpening(t *testing.T) {
	o := makeRandomOutput()
	msgChannel := make(chan interface{})
	d := testTransport{t, 0, 0, errorCounter{0, 100}, errorCounter{0, 1}, make(chan interface{}, 10), &sync.WaitGroup{}}
	ml := NewMemoryLogger()
	w := newMessageDispatcher(&o, &ml, msgChannel, &d)
	// Speed up for the test
	w.connectRetryMaxWaitTime = time.Millisecond * 10
	w.start()

	msgChannel <- "First message"

	select {
	case <-d.messageChan:
		t.Fatal("Should not receive a message!")
	// OK
	case <-time.After(50 * time.Millisecond):
	}

	// Close should be called automatically when it idles long enough
	w.stop()
	d.waitForClose()

}
