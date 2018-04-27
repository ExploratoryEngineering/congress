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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/logging"
	"github.com/eclipse/paho.mqtt.golang"
)

const (
	awsEndpoint   = model.TransportConfigKey("endpoint")
	awsClientID   = model.TransportConfigKey("clientid")
	awsClientCert = model.TransportConfigKey("clientCertificate")
	awsPrivateKey = model.TransportConfigKey("privateKey")
)

// awsiotTransport is similar to the MQTT type but uses a client certificate
// and a topic derived from the device EUI to publish messages.
//
// AWS IoT uses "shadows" to represent the state of the devices (aka "things")
// and these are maintained for retrieval. The thing shadows will use
// the same attributes as the ordinary device data messages.
//
// The endpoint, certificate and private key fields are required.
type awsiotTransport struct {
	endpoint   string
	clientid   string
	clientCert string
	privateKey string
	client     mqtt.Client
}

func init() {
	transports["awsiot"] = awsiotTransportFromConfig
}

func awsiotTransportFromConfig(tc model.TransportConfig) transport {
	if tc.String(awsEndpoint, "") == "" {
		return nil
	}
	if tc.String(awsClientCert, "") == "" {
		return nil
	}
	if tc.String(awsPrivateKey, "") == "" {
		return nil
	}
	return &awsiotTransport{
		endpoint:   tc.String(awsEndpoint, ""),
		clientid:   tc.String(awsClientID, "congress"),
		clientCert: tc.String(awsClientCert, ""),
		privateKey: tc.String(awsPrivateKey, ""),
	}
}

func (a *awsiotTransport) open(l *MemoryLogger) bool {
	var cert tls.Certificate
	var err error
	if cert, err = tls.X509KeyPair([]byte(a.clientCert+"\n"), []byte(a.privateKey+"\n")); err != nil {
		l.Append(NewLogEntry(err.Error()))
		logging.Warning("Unable to read the X509 key pair: %v", err)
		return false
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcps://%s:8883/mqtt", a.endpoint))
	opts.SetClientID(a.clientid)
	opts.SetMaxReconnectInterval(1 * time.Second)

	opts.SetTLSConfig(&tls.Config{
		Certificates: []tls.Certificate{cert},
	})

	a.client = mqtt.NewClient(opts)

	if token := a.client.Connect(); token.Wait() && token.Error() != nil {
		l.Append(NewLogEntry(token.Error().Error()))
		return false
	}
	return true
}

func (a *awsiotTransport) close(l *MemoryLogger) {
	if a.client == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			l.Append(NewLogEntry(fmt.Sprintf("Recovered from panic: %v", r)))
		}
	}()
	a.client.Disconnect(250)
}

// awsiotMessage is a wrapper just to format the message into something that
// AWS IoT accepts; a "state" object with a "desired" type inside to set
// the new state of the shadow
type awsiotMessage struct {
	State struct {
		Desired *deviceData `json:"desired"`
	} `json:"state"`
}

func (a *awsiotTransport) send(msg interface{}, logger *MemoryLogger) bool {
	dataMsg, ok := msg.(*PayloadMessage)
	if !ok {
		logging.Warning("Didn't receive a PayloadMessage type on channel but got %T. Dropping it.", msg)
		return true
	}

	dataOutput := newDeviceDataFromPayloadMessage(dataMsg)
	topicName := fmt.Sprintf("$aws/things/%s/shadow/update", dataOutput.DeviceEUI)
	qos := byte(1)
	retained := false

	var awsMsg awsiotMessage
	awsMsg.State.Desired = dataOutput
	messageBytes, err := json.Marshal(&awsMsg)
	if err != nil {
		logging.Warning("Unable to marshal AWS message: %v. Ignoring it.", err)
		return true
	}

	if token := a.client.Publish(topicName, qos, retained, messageBytes); token.Wait() && token.Error() != nil {
		logging.Info("Unable to forward message for device %s to AWS IoT: %v", dataOutput.DeviceEUI, token.Error())
		logger.Append(NewLogEntry(token.Error().Error()))
		return false
	}
	return true
}
