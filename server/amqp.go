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
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/ExploratoryEngineering/congress/logging"
	"github.com/ExploratoryEngineering/congress/model"
	"qpid.apache.org/amqp"
	"qpid.apache.org/electron"
)

//
// The AMQP transport config keys
const (
	amqpEndpoint      = model.TransportConfigKey("endpoint")
	amqpPort          = model.TransportConfigKey("port")
	amqpTLS           = model.TransportConfigKey("tls")
	amqpCertCheck     = model.TransportConfigKey("certCheck")
	amqpAllowInsecure = model.TransportConfigKey("allowInsecure")
	amqpUsername      = model.TransportConfigKey("username")
	amqpPassword      = model.TransportConfigKey("password")
	amqpContainerId   = model.TransportConfigKey("containerid")
	amqpAddress       = model.TransportConfigKey("address")
)

type amqpTransport struct {
	endpoint      string
	port          int
	useTLS        bool
	certCheck     bool
	allowInsecure bool
	username      string
	password      string
	connection    electron.Connection
	sender        electron.Sender
	containerId   string
	errors        chan LogEntry
	address       string
}

func init() {
	transports["amqp"] = amqpTransportFromConfig
}

// amqpTransportFromConfig creates a new AMQP transport if the supplied
// configuration is a AMQP configuration
func amqpTransportFromConfig(tc model.TransportConfig) transport {
	if tc.String(model.TransportConfigKey(model.TransportTypeKey), "") == "" {
		return nil
	}

	// this might be a AMQP config - decode and return it
	ret := amqpTransport{
		endpoint:      tc.String(amqpEndpoint, ""),
		username:      tc.String(amqpUsername, ""),
		password:      tc.String(amqpPassword, ""),
		useTLS:        tc.Bool(amqpTLS, false),
		certCheck:     tc.Bool(amqpCertCheck, true),
		allowInsecure: tc.Bool(amqpAllowInsecure, false),
		port:          tc.Int(amqpPort, 5672),
		containerId:   tc.String(amqpContainerId, "congress"),
		errors:        make(chan LogEntry, 5),
		address:       tc.String(amqpAddress, "congress"),
	}
	return &ret
}

func (m *amqpTransport) isValid() bool {
	if m.endpoint != "" && m.port != 0 {
		return true
	}
	return false
}

// Open opens the AMQP transport. If there's an error it will return false.
// Useful diagnostic messages may be logged to the supplied logger.
func (m *amqpTransport) open(l *MemoryLogger) bool {
	var opts []electron.ConnectionOption
	opts = append(opts, electron.SASLEnable())
	opts = append(opts, electron.SASLAllowedMechs("ANONYMOUS PLAIN"))

	// Using TLS means we can allow PLAIN mechanism
	if m.allowInsecure {
		opts = append(opts, electron.SASLAllowInsecure(m.allowInsecure))
	} else {
		opts = append(opts, electron.SASLAllowInsecure(m.useTLS))
	}

	if m.username != "" {
		opts = append(opts, electron.User(m.username))
	}

	if m.password != "" {
		opts = append(opts, electron.Password([]byte(m.password)))
	}

	uri := fmt.Sprintf("%s:%d", m.endpoint, m.port)

	var cont = electron.NewContainer(m.containerId)

	var c electron.Connection
	var err error
	if m.useTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: !m.certCheck,
		}
		n, err := tls.Dial("tcp", uri, tlsConfig)
		if err != nil {
			logging.Warning("Error connecting to %s: %v!", uri, err)
			return false
		}
		c, err = cont.Connection(n, opts...)
	} else {
		c, err = cont.Dial("tcp", uri, opts...)
	}

	if err != nil {
		logging.Warning("Error connecting to %s: %v!", uri, err)
		return false
	}

	s, err := c.Sender(electron.Target(m.address))
	if err != nil {
		logging.Warning("Error creating sender on %s to address %s: %v!", uri, m.address, err)
		return false
	}

	m.connection = c
	m.sender = s
	return true
}

// Close closes the transport. If there's an issue closing the transport
// diagnostic messages can be logged to the supplied logger
func (m *amqpTransport) close(l *MemoryLogger) {
	if m.connection == nil {
		return
	}
	m.connection.Close(nil)
}

// Send sends a message on the transport. If the send fails it will return
// false and log any useful diagnostic messages to the supplied logger
func (m *amqpTransport) send(msg interface{}, logger *MemoryLogger) bool {
	if m.sender == nil {
		return false
	}

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
	outcome := m.sender.SendSync(amqp.NewMessageWith(bytes))
	if outcome.Status != electron.Accepted {
		logging.Info("Unable to send message to AMQP server %s:%d: %v", m.endpoint, m.port, outcome.Error)
		logger.Append(NewLogEntry(outcome.Error.Error()))
		return false
	}
	return true
}
