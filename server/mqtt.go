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

//
// The MQTT transport config keys
const (
	mqttEndpoint  = model.TransportConfigKey("endpoint")
	mqttPort      = model.TransportConfigKey("port")
	mqttTLS       = model.TransportConfigKey("tls")
	mqttCertCheck = model.TransportConfigKey("certCheck")
	mqttUsername  = model.TransportConfigKey("username")
	mqttPassword  = model.TransportConfigKey("password")
	mqttClientiD  = model.TransportConfigKey("clientid")
	mqttTopicName = model.TransportConfigKey("topicName")
)

type mqttTransport struct {
	endpoint  string
	port      int
	useTLS    bool
	certCheck bool
	username  string
	password  string
	client    mqtt.Client
	clientID  string
	errors    chan LogEntry
	topicName string
}

func init() {
	transports["mqtt"] = mqttTransportFromConfig
}

// MQTTTransportFromConfig creates a new MQTT transport if the supplied
// configuration is a MQTT configuration
func mqttTransportFromConfig(tc model.TransportConfig) transport {
	if tc.String(model.TransportConfigKey(model.TransportTypeKey), "") == "" {
		return nil
	}

	// this might be a MQTT config - decode and return it
	ret := mqttTransport{
		endpoint:  tc.String(mqttEndpoint, ""),
		username:  tc.String(mqttUsername, ""),
		password:  tc.String(mqttPassword, ""),
		useTLS:    tc.Bool(mqttTLS, false),
		certCheck: tc.Bool(mqttCertCheck, true),
		port:      tc.Int(mqttPort, 1883),
		clientID:  tc.String(mqttClientiD, "congress"),
		errors:    make(chan LogEntry, 5),
		topicName: tc.String(mqttTopicName, "congress"),
	}
	return &ret
}

func (m *mqttTransport) isValid() bool {
	if m.endpoint != "" && m.port != 0 {
		return true
	}
	return false
}

func (m *mqttTransport) connect(l *MemoryLogger) bool {
	token := m.client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		l.Append(NewLogEntry(err.Error()))
		return false
	}
	return true
}

// Open opens the MQTT transport. If there's an error it will return false.
// Useful diagnostic messages may be logged to the supplied logger.
func (m *mqttTransport) open(l *MemoryLogger) bool {
	opts := mqtt.NewClientOptions()
	proto := "tcp"
	if m.useTLS {
		proto = "ssl"
	}
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", proto, m.endpoint, m.port))
	opts.SetClientID(m.clientID)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetWriteTimeout(1 * time.Second)
	// If the client auto reconnects it will block until a connection becomes
	// available. This isn't very helpful if the messages is going to be
	// queued and the decoding pipeline might drop messages that aren't processed
	// quickly enough by the clients. The backlog will keep up to 50 messages
	// in memory until they are discarded.
	opts.SetAutoReconnect(false)
	opts.SetMessageChannelDepth(1)
	opts.SetCleanSession(true)
	if m.username != "" {
		opts.SetUsername(m.username)
	}
	if m.password != "" {
		opts.SetPassword(m.password)
	}
	if m.useTLS {
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: m.certCheck,
		})
	}

	m.client = mqtt.NewClient(opts)

	return m.connect(l)
}

// Close closes the transport. If there's an issue closing the transport
// diagnostic messages can be logged to the supplied logger
func (m *mqttTransport) close(l *MemoryLogger) {
	if m.client == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			l.Append(NewLogEntry(fmt.Sprintf("Recovered from panic: %v", r)))
		}
	}()
	m.client.Disconnect(250)
}

// Send sends a message on the transport. If the send fails it will return
// false and log any useful diagnostic messages to the supplied logger
func (m *mqttTransport) send(msg interface{}, logger *MemoryLogger) bool {
	if m.client == nil {
		return false
	}
	if !m.client.IsConnected() {
		// Attempt a reconnect
		m.connect(logger)
	}
	qos := byte(1)
	retained := false

	// Payload is an data structure. Convert into same format as the websocket
	// output (apiDeviceData) and pass on.
	dataMsg, ok := msg.(*PayloadMessage)
	if !ok {
		logging.Warning("Didn't receive a PayloadMessage type on channel but got %T. Silently dropping it.", msg)
		return true
	}
	dataOutput := newDeviceDataFromPayloadMessage(dataMsg)
	bytes, err := json.Marshal(dataOutput)
	if err != nil {
		logging.Warning("Unable to marshal %T into JSON: %v. Silently dropping it.", dataMsg, err)
		return true
	}
	token := m.client.Publish(m.topicName, qos, retained, bytes)
	token.Wait()
	if err := token.Error(); err != nil {
		logging.Info("Unable to send message to MQTT server %s:%d: %v", m.endpoint, m.port, err)
		logger.Append(NewLogEntry(err.Error()))
		return false
	}
	return true
}
