//+build amqp

package server

//
//Copyright 2018 Ulf Lilleengen
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
	"net"
	"path"
	"runtime"
	"testing"

	"github.com/ExploratoryEngineering/congress/model"
	//"github.com/ExploratoryEngineering/congress/logging"
	"github.com/ExploratoryEngineering/congress/utils"
	"qpid.apache.org/electron"
)

// Stolen from electron_test
func fatalIf(t *testing.T, err error) {
	if err != nil {
		_, file, line, ok := runtime.Caller(1) // annotate with location of caller.
		if ok {
			_, file = path.Split(file)
		}
		t.Fatalf("(from %s:%d) %v", file, line, err)
	}
}

// Start a server, return listening addr and channel for incoming Connections.
func newServer(t *testing.T, cont electron.Container, port int, opts ...electron.ConnectionOption) <-chan electron.Connection {
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	fatalIf(t, err)
	ch := make(chan electron.Connection)
	go func() {
		conn, err := listener.Accept()
		c, err := cont.Connection(conn, append([]electron.ConnectionOption{electron.Server()}, opts...)...)
		fatalIf(t, err)
		ch <- c
	}()
	return ch
}

func launchAmqpServer(t *testing.T, port int) <-chan electron.Connection {
	return newServer(t, electron.NewContainer("test-server"), port)
}

var amqpConfig = `
	{
		"type": "amqp",
		"endpoint": "localhost",
		"port": 5672,
		"tls": false,
		"certCheck": false,
        "containerId": "test-client",
		"allowInsecure": true
	}
`

// Happy path testing - open connection, send message, close connection
func TestAMQPTransport(t *testing.T) {
	tc, err := model.NewTransportConfig(amqpConfig)
	if err != nil {
		t.Fatal("Got error parsing transport")
	}
	transport := amqpTransportFromConfig(tc).(*amqpTransport)
	transport.port, _ = utils.FreePort()
	if !transport.isValid() {
		t.Fatal("Transport isn't valid")
	}
	ch := launchAmqpServer(t, transport.port)
	if transport == nil {
		t.Fatal("Configuration isn't recognized as AMQP config")
	}

	ml := NewMemoryLogger()
	if transport.open(&ml) == false {
		for _, v := range ml.Items() {
			if v.IsValid() {
				t.Logf("%s: %s", v.TimeString(), v.Message)
			}
		}
		t.Fatal("Could not open transport!")
	}

	server := <-ch
	rchan := make(chan electron.Receiver, 1)
	go func() {
		for in := range server.Incoming() {
			switch in := in.(type) {
			case *electron.IncomingReceiver:
				in.SetCapacity(100)
				in.SetPrefetch(true)
				rchan <- in.Accept().(electron.Receiver)
			default:
				in.Accept()
			}
		}
	}()

	receiver := <-rchan

	if receiver.Target() != "congress" {
		t.Fatal("Not attached to expected address")
	}

	for i := 0; i < 100; i++ {

		sendDone := make(chan struct{})
		go func() {
			defer close(sendDone)
			if transport.send(makePayloadMessage(), &ml) == false {
				t.Fatal("Could not send message on transport")
			}
		}()

		rm, err := receiver.Receive()
		if err != nil {
			t.Fatal(err)
		}
		rm.Accept()
		<-sendDone
	}
	for _, v := range ml.Items() {
		if v.IsValid() {
			t.Logf("%s: %s", v.TimeString(), v.Message)
		}
	}
	transport.close(&ml)
	server.Close(nil)
}
