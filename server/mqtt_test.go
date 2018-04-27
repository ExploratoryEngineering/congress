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

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/utils"
	"github.com/ExploratoryEngineering/logging"
	"github.com/surgemq/surgemq/service"
)

/*
 Additional dependencies not explicitly mentioned:
	- github.com/surgemq/surgemq
  	- github.com/surge/glog
	- github.com/surgemq/message
*/

// Test the MQTT transport. In time this might evolve into a generic transport
// test that can be applied to multiple transports.
var svr *service.Server

func launchServer(t *testing.T, port int, authMock string) {
	svr = &service.Server{
		SessionsProvider: "mem",
		TopicsProvider:   "mem",
		Authenticator:    authMock,
	}
	mqttaddr := fmt.Sprintf("tcp://localhost:%d", port)
	go func() {
		if err := svr.ListenAndServe(mqttaddr); err != nil {
			t.Fatalf("Got error running MQTT broker: %v", err)
		}
	}()
}

func stopServer(t *testing.T) {
	if err := svr.Close(); err != nil {
		t.Fatal("Got error shutting down MQTT broker: ", err)
	}
}

var mqttConfig = `
	{
		"type": "mqtt",
		"endpoint": "localhost",
		"port": 1883,
		"tls": false,
		"certCheck": false,
		"username": "user1",
		"password": "user1",
		"clientid": "congress",
		"topicName": "mqtt-transport-test"
	}
`

func makePayloadMessage() *PayloadMessage {
	return &PayloadMessage{
		Device:      model.NewDevice(),
		Application: model.NewApplication(),
		FrameContext: FrameContext{
			Device:         model.NewDevice(),
			Application:    model.NewApplication(),
			GatewayContext: GatewayPacket{},
		},
	}
}

// Happy path testing - open connection, send message, close connection
func TestMQTTTransport(t *testing.T) {
	tc, err := model.NewTransportConfig(mqttConfig)
	if err != nil {
		t.Fatal("Got error parsing transport")
	}
	transport := mqttTransportFromConfig(tc).(*mqttTransport)
	transport.port, _ = utils.FreePort()
	if !transport.isValid() {
		t.Fatal("Transport isn't valid")
	}
	launchServer(t, transport.port, "mockSuccess")
	defer stopServer(t)
	if transport == nil {
		t.Fatal("Configuration isn't recognized as MQTT config")
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
	for i := 0; i < 100; i++ {

		if transport.send(makePayloadMessage(), &ml) == false {
			t.Fatal("Could not send message on transport")
		}

	}
	for _, v := range ml.Items() {
		if v.IsValid() {
			t.Logf("%s: %s", v.TimeString(), v.Message)
		}
	}
	transport.close(&ml)
}

// Test authentication/connection failures
func TestMQTTAuthFail(t *testing.T) {
	tc, err := model.NewTransportConfig(mqttConfig)
	if err != nil {
		t.Fatal("Got error parsing transport")
	}
	transport := mqttTransportFromConfig(tc).(*mqttTransport)
	transport.port, _ = utils.FreePort()
	launchServer(t, transport.port, "mockFailure")

	defer stopServer(t)
	if transport == nil {
		t.Fatal("Configuration isn't recognized as MQTT config")
	}

	ml := NewMemoryLogger()
	if transport.open(&ml) != false {
		t.Fatal("Expected failure when opening transport")
	}
	for _, v := range ml.Items() {
		if v.IsValid() {
			logging.Debug("log entry %s: %s", v.TimeString(), v.Message)
		}
	}
	transport.close(&ml)
}

func TestMQTTSendError(t *testing.T) {
	tc, err := model.NewTransportConfig(mqttConfig)
	if err != nil {
		t.Fatal("Got error parsing transport")
	}
	transport := mqttTransportFromConfig(tc)
	if transport == nil {
		t.Fatal("Configuration isn't recognized as MQTT config")
	}

	ml := NewMemoryLogger()

	for i := 0; i < 10; i++ {
		if transport.send(PayloadMessage{}, &ml) != false {

		}
	}
	for _, v := range ml.Items() {
		if v.IsValid() {
			logging.Debug("%s: %s", v.TimeString(), v.Message)
		}
	}
	transport.close(&ml)
}

// TODO: Test multiple clients, verify output, TLS, cert checks
// ...invalid certs, connection errors, send errors, surgemQ client
///
