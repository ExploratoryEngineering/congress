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
	"math/rand"
	"sync"
	"time"

	"fmt"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/logging"
)

// This is the implementation of the message dispatcher process that grabs
// messages and forwards to a destination output. There's just one destination
// output at this time (MQTT) but more will probably follow Real Soon Now
// (AQMP, gRPC, TCP, webhooks, AWS IoT... possibly even UDP if someone wears
// their crazy hat for long enough)
//
// The dispatcher uses one messaging channel to receive messages that are sent
// and one channel to control it. When a termination signal is sent to the
// dispatcher the connection will be dropped and no more messages will be sent.
//

type dispatcherState string

const (
	// The length of the backlog.
	maxBacklogLength = 10
	maxRetries       = 10
)
const (
	dispatcherOpening = dispatcherState("opening")
	dispatcherIdle    = dispatcherState("idle")
	dispatcherActive  = dispatcherState("active")
)

// Enter idle state after 10 minutes
const (
	idleTime            = time.Second * 600
	maxWaitConnectRetry = time.Second * 60
	connectRetryTime    = time.Second * 1
	sendRetryTimeMs     = 1000
)

func newMessageDispatcher(op *model.AppOutput, ml *MemoryLogger, queue <-chan interface{}, destination transport) *messageDispatcher {
	return &messageDispatcher{
		dispatcherState:         dispatcherOpening,
		op:                      op,
		messages:                queue,
		logger:                  ml,
		destination:             destination,
		terminate:               make(chan bool),
		backlog:                 make(chan backlogMessage, maxBacklogLength),
		sendRetryTimeMs:         sendRetryTimeMs,
		connectRetryTime:        connectRetryTime,
		connectRetryMaxWaitTime: maxWaitConnectRetry,
		idleTime:                idleTime,
		mutex:                   &sync.Mutex{},
	}
}

// Backlog messages. These messages will be retried N times.
type backlogMessage struct {
	msg     interface{}
	retries int
}

// messageDispatcher is responsible for forwarding data to a single transport
type messageDispatcher struct {
	op                      *model.AppOutput
	messages                <-chan interface{}
	logger                  *MemoryLogger
	terminate               chan bool
	destination             transport
	dispatcherState         dispatcherState
	eventRouter             router
	backlog                 chan backlogMessage
	sendRetryTimeMs         int           // for testing
	connectRetryTime        time.Duration // for testing
	connectRetryMaxWaitTime time.Duration // for testing
	maxIdleTime             time.Duration // for testing
	idleTime                time.Duration // for testing
	mutex                   *sync.Mutex
}

func (o *messageDispatcher) closeTransport() {
	if o.state() == dispatcherActive {
		o.destination.close(o.logger)
		o.setState(dispatcherIdle)
	}
}

func (o *messageDispatcher) sendMessage(msg interface{}, retries int) {
	// make sure the connection is open
	if o.state() != dispatcherActive {
		o.setState(dispatcherIdle)
		sleepTime := o.connectRetryTime
		for o.destination.open(o.logger) != true {
			// Keep trying until dispatcher is terminated or connection succeeds
			select {
			case <-o.terminate:
				return
			case <-time.After(sleepTime):
				// Throttle back on connection attempts until there's one retry every
				// 60 seconds
				sleepTime *= 2
				if sleepTime > o.connectRetryMaxWaitTime {
					sleepTime = o.connectRetryMaxWaitTime
				}
			}
		}
		o.setState(dispatcherActive)
		o.logger.Append(NewLogEntry("Connected"))
	}
	// Process message
	if !o.destination.send(msg, o.logger) {
		// requeue the message
		if retries > maxRetries {
			logging.Warning("Dropping message to %s after %d retries. Message = %v", o.op.EUI, retries-1, msg)
			o.logger.Append(NewLogEntry(fmt.Sprintf("Unable to send after %d retries. Message has been dropped", maxRetries)))
			return
		}
		msg := fmt.Sprintf("Send failed, re-queuing message to %s (%d of %d retries)", o.op.EUI.String(), retries+1, maxRetries)
		o.logger.Append(NewLogEntry(msg))
		o.backlog <- backlogMessage{msg: msg, retries: retries + 1}
		<-time.After(time.Duration(rand.Intn(o.sendRetryTimeMs)) * time.Millisecond)
	}
}

// Main loop for the message dispatcher. Wait for either terminate messages
// or messages to be forwarded. If the dispatcher have been idle for too long
// close the connection.
func (o *messageDispatcher) dispatcherLoop() {
	defer o.closeTransport()

	// Keep forwarding messages
	for {
		select {
		case <-o.terminate:
			close(o.terminate)
			return
		case msg := <-o.backlog:
			o.sendMessage(msg.msg, msg.retries)
		case msg, ok := <-o.messages:
			if !ok {
				logging.Debug("Message channel is closed! Terminating!")
				o.logger.Append(NewLogEntry("Disabled internally"))
				o.closeTransport()
				return
			}
			o.sendMessage(msg, 0)
		case <-time.After(o.idleTime):
			if o.state() == dispatcherActive {
				o.logger.Append(NewLogEntry("Entering idle state"))
				o.closeTransport()
			}
		}
	}
}

// Start the dispatcher. This logs an entry into the log
func (o *messageDispatcher) start() error {
	logging.Debug("Starting dispatcher for output %s", o.op.EUI)
	o.logger.Append(NewLogEntry("Started"))
	o.setState(dispatcherIdle)
	go o.dispatcherLoop()
	return nil
}

func (o *messageDispatcher) stop() error {
	logging.Debug("Stopping dispatcher for output %s", o.op.EUI)
	select {
	case o.terminate <- true:
		o.logger.Append(NewLogEntry("Stopped"))
	case <-time.After(100 * time.Millisecond):
		logging.Info("Dispatcher with EUI %s didn't respond - skipping", o.op.EUI)
	}
	return nil
}

func (o *messageDispatcher) logs() *MemoryLogger {
	return o.logger
}

func (o *messageDispatcher) messageChannel() <-chan interface{} {
	return o.messages
}

func (o *messageDispatcher) output() *model.AppOutput {
	return o.op
}

func (o *messageDispatcher) status() string {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return string(o.dispatcherState)
}

// The state field is accessed by multiple goroutines, both in the http server
// and via the output manager. Using mutex here.
func (o *messageDispatcher) setState(status dispatcherState) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.dispatcherState = status
}

// The state accessor - used by multiple goroutines
func (o *messageDispatcher) state() dispatcherState {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.dispatcherState
}
