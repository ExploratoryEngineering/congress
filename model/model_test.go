package model

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
	"encoding/json"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/protocol"
)

// This is borderline gaming the system but the code should at least run through once
func TestAppNonceGenerator(t *testing.T) {
	app := Application{Tags: NewTags()}
	app.GenerateAppNonce()
}

func TestDeviceStateConversion(t *testing.T) {
	states := []DeviceState{
		OverTheAirDevice,
		PersonalizedDevice,
		DisabledDevice,
	}
	for _, v := range states {
		if val, err := DeviceStateFromString(v.String()); val != v || err != nil {
			t.Errorf("Coudldn't convert %v to and from string (error is %v)", v, err)
		}
	}

	if _, err := DeviceStateFromString("unknown state"); err == nil {
		t.Error("Expected error when using unknown string format")
	}
}

func TestRXWindows(t *testing.T) {
	// These values are hard coded. The *real* test will use the device's settings
	device := Device{}
	if device.GetRX1Window() != (time.Second * 1) {
		t.Error("Someone must have fixed the GetRX1Window func but not the test")
	}
	if device.GetRX2Window() != (time.Second * 2) {
		t.Error("Someone must have fixed the GetRX2Window func but not the test")
	}
}

func TestDevNonce(t *testing.T) {
	d := Device{
		DevNonceHistory: []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}
	if !d.HasDevNonce(1) || !d.HasDevNonce(5) || !d.HasDevNonce(9) {
		t.Fatal("Expected 1, 5 and 9 to be in nonce history")
	}
	if d.HasDevNonce(0) || d.HasDevNonce(10) {
		t.Fatal("Didn't expect 0 or 10 to be in nonce history")
	}
}

func TestDeviceDataCompare(t *testing.T) {
	d1 := DeviceData{DeviceEUI: protocol.EUIFromUint64(0), Data: []byte{1, 2, 3}, Frequency: 99.0, GatewayEUI: protocol.EUIFromUint64(1)}
	d2 := DeviceData{DeviceEUI: protocol.EUIFromUint64(1), Data: []byte{1, 2, 3}, Frequency: 98.0, GatewayEUI: protocol.EUIFromUint64(1)}
	d3 := DeviceData{DeviceEUI: protocol.EUIFromUint64(0), Data: []byte{1, 2, 3}, Frequency: 99.0, GatewayEUI: protocol.EUIFromUint64(1)}

	if d1.Equals(d2) || d2.Equals(d1) {
		t.Fatal("Should not be the same")
	}

	if !d1.Equals(d3) || !d3.Equals(d1) {
		t.Fatal("Should be equal")
	}
}

func TestGatewayCompare(t *testing.T) {
	g1 := Gateway{GatewayEUI: protocol.EUIFromUint64(0), IP: net.ParseIP("127.0.0.1"), Altitude: 1}
	g2 := Gateway{GatewayEUI: protocol.EUIFromUint64(1), IP: net.ParseIP("127.1.0.1"), Altitude: 2}
	g3 := Gateway{GatewayEUI: protocol.EUIFromUint64(0), IP: net.ParseIP("127.0.0.1"), Altitude: 1}

	if g1.Equals(g2) || g2.Equals(g1) {
		t.Fatal("Should not be the same")
	}

	if !g1.Equals(g3) || !g3.Equals(g1) {
		t.Fatal("Should be equal")
	}

	g4 := NewGateway()
	g5 := NewGateway()

	if !g4.Equals(g5) {
		t.Fatal("Empty gateways should be equal")
	}
	g4.Tags.SetTag("foo", "bar")
	g5.Tags.SetTag("foo", "baz")

	if g4.Equals(g5) || g5.Equals(g4) {
		t.Fatal("Gateways shouldn' be equal at this time")
	}
}

func TestNewAPIToken(t *testing.T) {
	readonlyRoot, err := NewAPIToken("0", "/", true)
	if err != nil {
		t.Fatal("Got error generating token: ", err)
	}
	if readonlyRoot.Write != true {
		t.Fatal("Expected write token in return")
	}
	if readonlyRoot.Resource != "/" {
		t.Fatal("Expected root token")
	}
	if readonlyRoot.UserID != "0" {
		t.Fatal("Expected user ID to be 0")
	}

	const tokenCount = 1000
	t.Logf("Generating %d tokens...", tokenCount)
	// Make 100 tokens and ensure they're all different
	tokens := make(map[string]APIToken)
	for i := 0; i < tokenCount; i++ {
		newToken, err := NewAPIToken("0", "/foo", false)
		if err != nil {
			t.Fatal("Got error generating token: ", err)
		}
		t.Logf("T=%s", newToken.Token)
		tokens[newToken.Token] = newToken
	}
	if len(tokens) != tokenCount {
		t.Fatalf("Did not get the expected number of tokens. Got %d, expected %d", len(tokens), tokenCount)
	}
}

func TestTransportConfig(t *testing.T) {
	tc, err := NewTransportConfig("{}")
	if err != nil {
		t.Fatalf("Got error creating transport config: %v", err)
	}
	if err := json.Unmarshal([]byte(`{"string":"string", "int":99, "bool":true}`), &tc); err != nil {
		t.Fatal("Got error unmarshaling: ", err)
	}

	if tc.String(TransportConfigKey("string"), "foo") != "string" {
		t.Fatal("Expected string value")
	}
	if tc.Bool(TransportConfigKey("bool"), false) != true {
		t.Fatal("Expected bool value")
	}
	if tc.Int(TransportConfigKey("int"), -1) != 99 {
		t.Fatalf("Expected int 99 but got %d", tc.Int(TransportConfigKey("int"), -1))
	}

	if tc.String(TransportConfigKey("foo"), "bar") != "bar" {
		t.Fatal("Did not expect foo string")
	}
	if tc.Bool(TransportConfigKey("foo"), true) != true {
		t.Fatal("Did not expect foo bool")
	}
	if tc.Int(TransportConfigKey("foo"), -1) != -1 {
		t.Fatal("Did not expect foo int")
	}
}

// Tests are test. Just running through the code helps.
func TestAppOutput(t *testing.T) {
	NewAppOutput()
}

func TestDownstreamMessage(t *testing.T) {
	eui := protocol.EUIFromUint64(0x0abcdef0)
	msg := NewDownstreamMessage(eui, 100)
	if msg.CreatedTime == 0 {
		t.Fatal("Expected created time to be set")
	}

	if msg.IsComplete() || msg.State() != UnsentState {
		t.Fatal("Message should not be completed and in unsent state")
	}

	// No ack and message is sent: not pending
	msg.SentTime = time.Now().Unix()
	if !msg.IsComplete() || msg.State() != SentState {
		t.Fatal("Expected message to be completed and in sent state")
	}

	// Set ack flag. The state should become pending
	msg.Ack = true
	if msg.IsComplete() {
		t.Fatal("Expected message not to be completed")
	}

	msg.AckTime = time.Now().Unix()
	if !msg.IsComplete() || msg.State() != AcknowledgedState {
		t.Fatal("Expected message to be completed and acknowledged state")
	}

	msg.Data = "010203040506070809"
	if !reflect.DeepEqual(msg.Payload(), []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}) {
		t.Fatal("Not the payload I expected")
	}

	msg.Data = "Random characters"
	if len(msg.Payload()) != 0 {
		t.Fatal("Expected empty payload")
	}
}
