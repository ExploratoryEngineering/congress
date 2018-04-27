package gateway

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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage/memstore"
	"github.com/ExploratoryEngineering/congress/utils"
	"github.com/ExploratoryEngineering/pubsub"
)

type serverConfig struct {
	forwarder *GenericPacketForwarder
	clientUDP *net.UDPConn
}

var gwStorage = memstore.NewMemoryGatewayStorage()

func setupServer(t *testing.T) serverConfig {
	ret := serverConfig{}

	port, err := utils.FreePort()
	if err != nil {
		t.Fatal("Could not allocate free port: ", err)
	}

	router := pubsub.NewEventRouter(5)
	context := server.Context{GwEventRouter: &router, Config: &server.Configuration{}}
	ret.forwarder = NewGenericPacketForwarder(port, gwStorage, &context)

	go ret.forwarder.Start()

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatal("Couldn't resolve server UDP: ", err)
	}

	ret.clientUDP, err = net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		t.Fatal("Got error dialing to server: ", err)
	}

	return ret
}

func (s *serverConfig) close() {
	<-time.After(100 * time.Millisecond)
	s.clientUDP.Close()
}

func getValidRxPk(data string) Rxpk {
	return Rxpk{
		Time:                "2017-02-01T23:55:55.233Z",
		Timestamp:           2017,
		Frequency:           868.1,
		ConcentratorChannel: 1,
		ConcentratorRFChain: 0,
		ModulationID:        "lora",
		DataRateID:          "DR7BW12",
		CodingRateID:        "3/4",
		RSSI:                12,
		LoraSNRRatio:        20,
		PayloadSize:         12,
		RFPackets:           data,
	}

}

// A simple mock-up of the gateway forwarder that pushes various packets
// via UDP. It uses the protocol type to encode and decode packets.
func TestGenericPacketForwarder(t *testing.T) {
	s := setupServer(t)
	defer s.close()

	sampleEUI := protocol.EUIFromUint64(0x0102030405060708)
	gwStorage.Put(model.Gateway{
		GatewayEUI: sampleEUI,
		StrictIP:   false,
		Tags:       model.NewTags(),
	}, model.SystemUserID)
	pullData := GwPacket{Token: 0x0102, Identifier: PullData, GatewayEUI: sampleEUI}
	buf, err := pullData.MarshalBinary()
	if err != nil {
		t.Fatal("Got error marshaling message: ", err)
	}
	_, err = s.clientUDP.Write(buf)
	if err != nil {
		t.Fatal("Got error sending PULL_DATA to server: ", err)
	}

	radioMessage := "Thisisthedatafromthegateway"
	encodedBytes := base64.StdEncoding.EncodeToString([]byte(radioMessage))

	// Write a PUSH_DATA packet with JSON data. This should yield an output
	rxdata := RXData{
		Data: []Rxpk{getValidRxPk(encodedBytes), getValidRxPk("some other")},
	}
	jsonBuffer, _ := json.Marshal(rxdata)
	pushData := GwPacket{Token: 0x0202, Identifier: PushData, GatewayEUI: sampleEUI, JSONString: string(jsonBuffer)}

	buf, _ = pushData.MarshalBinary()
	_, err = s.clientUDP.Write(buf)
	if err != nil {
		t.Fatal("Got error sending PUSH_DATA to server: ", err)
	}

	select {
	case receivedPacket := <-s.forwarder.Output():
		if receivedPacket.Gateway.GatewayClock != 2017 {
			if string(receivedPacket.RawMessage) != radioMessage {
				t.Fatal("Got different message than what was sent.")
			}
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Did not get a packet for 1 second")
	}

	// Pretend a send to the gateway. We won't bother with the output here.
	sendMessage := "Thisisthemessagefromtheserver"
	s.forwarder.Input() <- server.GatewayPacket{
		RawMessage: []byte(sendMessage),
		Radio:      server.RadioContext{},
		Gateway:    server.GatewayContext{},
	}
}

// Set up the gateway and pepper it with random data. This will happen.
func TestInvalidBinaryData(t *testing.T) {
	s := setupServer(t)
	defer s.close()

	buf := make([]byte, 0)
	_, err := s.clientUDP.Write(buf)

	buf = make([]byte, 4096)
	for i := 0; i < 100; i++ {
		rand.Read(buf)
		_, err = s.clientUDP.Write(buf)
		if err != nil {
			t.Fatal("Got error writing packet: ", err)
		}
	}
}

// Pepper the gateway with partial random data.
func TestInvalidPacketData(t *testing.T) {
	s := setupServer(t)
	defer s.close()

	buf := make([]byte, 0)
	s.clientUDP.Write(buf)

	buf = make([]byte, 1024)
	for i := 0; i < 10; i++ {
		rand.Read(buf)
		for packetType := byte(0); packetType < 100; packetType++ {
			buf[0] = packetType
			s.clientUDP.Write(buf)
		}
	}

	select {
	case <-s.forwarder.Output():
		t.Fatal("Did not expect output on channel")
	case <-time.After(100 * time.Millisecond):
		// OK.
	}
}

// Send valid PUSH data but with invalid JSON, both partial and complete
func TestInvalidPushData(t *testing.T) {
	s := setupServer(t)
	defer s.close()

	invalidJSON := []string{
		"{}",
		`{"rxpk": {}}`,
		`{"rxpk": []}`,
		`{"rxpk": [{"data": "Just some random data"}, "stat": {}}]`,
	}

	for _, str := range invalidJSON {
		pk := GwPacket{ProtocolVersion: 0, Token: 0, Identifier: PullData, JSONString: str}
		buf, err := pk.MarshalBinary()
		if err != nil {
			t.Fatal("Got error marshaling packet: ", err)
		}
		_, err = s.clientUDP.Write(buf)
		if err != nil {
			t.Fatal("Got error writing packet: ", err)
		}
	}

}

// Test filtering based on source EUI (with non-strict IP checks) then with
// strict IP checks.
func TestEUIFilteringOnEUI(t *testing.T) {
	s := setupServer(t)
	defer s.close()

	sendMessage := func(eui protocol.EUI) {
		radioMessage := "Thisisthedatafromthegateway"
		encodedBytes := base64.StdEncoding.EncodeToString([]byte(radioMessage))

		// Write a PUSH_DATA packet with JSON data. This should yield an output
		rxdata := RXData{
			Data: []Rxpk{getValidRxPk(encodedBytes)},
		}

		jsonBuffer, _ := json.Marshal(rxdata)
		pushData := GwPacket{Token: 0x0202, Identifier: PushData, GatewayEUI: eui, JSONString: string(jsonBuffer)}

		buf, err := pushData.MarshalBinary()
		if err != nil {
			t.Fatal("Got error marshaling packet: ", err)
		}
		_, err = s.clientUDP.Write(buf)
		if err != nil {
			t.Fatal("Got error writing packet: ", err)
		}
	}

	validEUIvalidIP := protocol.EUIFromUint64(0x0102030405060708)
	invalidEUI := protocol.EUIFromUint64(0x0807060504030201)
	validEUIinvalidIP := protocol.EUIFromUint64(0x0807060504030202)

	gwStorage.Put(model.Gateway{
		GatewayEUI: validEUIvalidIP,
		StrictIP:   false,
		IP:         net.ParseIP("127.0.0.1"),
		Tags:       model.NewTags(),
	}, model.SystemUserID)

	gwStorage.Put(model.Gateway{
		GatewayEUI: validEUIinvalidIP,
		StrictIP:   true,
		IP:         net.ParseIP("127.10.10.1"),
		Tags:       model.NewTags(),
	}, model.SystemUserID)

	sendMessage(validEUIvalidIP)
	select {
	case <-s.forwarder.Output():
		// OK - expect message here
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Did not receive a message from a valid IP")
	}

	sendMessage(validEUIinvalidIP)
	select {
	case <-s.forwarder.Output():
		t.Fatal("Expected message with invalid IP to be rejected")
	case <-time.After(100 * time.Millisecond):
		// OK - should be rejected
	}

	sendMessage(invalidEUI)
	select {
	case <-s.forwarder.Output():
		t.Fatal("Expected message with invalid EUI to be rejected")
	case <-time.After(100 * time.Millisecond):
		// OK - should be rejected
	}

}
