package main

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
	"math/rand"
	"net"
	"time"

	"encoding/base64"
	"encoding/json"

	cgw "github.com/ExploratoryEngineering/congress/gateway"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/logging"
)

// SyntheticForwarder is a synthetic packet forwarder that exposes two channels
// to the outside -- one for sending messages and one for received messages.
// Both channels operate on simple byte arrays. They will be wrapped into
// the appropriate JSON + packet forwarder structs and forwarded to the backend.
type SyntheticForwarder struct {
	messageChannel  chan string
	shutdownChannel chan bool
	gatewayEUI      string
	host            string
	port            int
	upstreamConn    *net.UDPConn
	downstreamConn  *net.UDPConn
	outputChannel   chan []byte
	incomingMessage chan []byte
}

// NewSyntheticForwarder creates a new synthetic packet forwarder
func NewSyntheticForwarder(messageChannel chan string, shutdownChannel chan bool, gatewayEUI string, host string, port int) *SyntheticForwarder {

	return &SyntheticForwarder{
		messageChannel:  messageChannel,
		shutdownChannel: shutdownChannel,
		gatewayEUI:      gatewayEUI,
		host:            host,
		port:            port,
		outputChannel:   make(chan []byte),
		incomingMessage: make(chan []byte),
	}
}

func (g *SyntheticForwarder) sendPullData() {
	// 1. Send a PULL_DATA to Congress, in order to communicate Gateway gatewayEUI

	eui, _ := protocol.EUIFromString(g.gatewayEUI)
	packet := cgw.GwPacket{
		ProtocolVersion: 0,
		Token:           0,
		Identifier:      2, // PULL_DATA
		GatewayEUI:      eui,
	}

	buf, err := packet.MarshalBinary()
	if err != nil {
		logging.Warning("Unable to marshal PULL_DATA: %v", err)
		return
	}
	if _, err := g.downstreamConn.Write(buf); err != nil {
		logging.Warning("Error writing PULL_DATA packet : ", err)
	}
}

func (g *SyntheticForwarder) initUDP() bool {
	upServerAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", g.host, g.port))
	if err != nil {
		logging.Error("Error resolving target address for Congress (%s:%d): %v", g.host, g.port, err)
		return false
	}
	upLocalAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		logging.Error("Error resolving ':0' %v", err)
		return false
	}
	g.upstreamConn, err = net.DialUDP("udp", upLocalAddr, upServerAddr)
	if err != nil {
		logging.Error("DialUDP error: %v", err)
		return false
	}

	downServerAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", g.host, g.port))
	if err != nil {
		logging.Error("Error resolving downstream UDP address: %v", err)
		return false
	}
	downLocalAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		logging.Error("Error resolving downstream :0 %v", err)
		return false
	}
	g.downstreamConn, err = net.DialUDP("udp", downLocalAddr, downServerAddr)
	if err != nil {
		logging.Error("Error opening downstream UDP: %v", err)
		return false
	}
	// 1. Send a PULL_DATA to Congress, in order to communicate Gateway gatewayEUI + destination for
	// downstream messages
	g.sendPullData()

	return true
}

func (g *SyntheticForwarder) forwarder(shutdownChannel chan bool) {
	eui, _ := protocol.EUIFromString(g.gatewayEUI)

	// 2. Sent PUSH_DATA packets from devices
	var timestamp uint32
	for {
		select {
		case message := <-g.messageChannel:
			timestamp++

			rxPacket := cgw.Rxpk{
				Time:                time.Now().UTC().Format("2006-01-02T15:04:05-0700"),
				Timestamp:           timestamp,
				Frequency:           868.1,
				ConcentratorChannel: 1,
				ConcentratorRFChain: 0,
				ModulationID:        "lora",
				DataRateID:          "SF12BW125",
				CodingRateID:        "3/4",
				RSSI:                12,
				LoraSNRRatio:        20,
				PayloadSize:         uint32(len(message)),
				RFPackets:           message,
			}

			rxData := cgw.RXData{Data: make([]cgw.Rxpk, 1)}
			rxData.Data[0] = rxPacket

			data, err := json.Marshal(rxData)
			if err != nil {
				logging.Warning("Unable to marshal JSON data: ", err)
				continue
			}

			packet := cgw.GwPacket{
				ProtocolVersion: 0,
				Token:           uint16(rand.Intn(0xFFFF)),
				Identifier:      cgw.PushData,
				JSONString:      string(data),
				GatewayEUI:      eui,
			}

			buf, err := packet.MarshalBinary()
			if err != nil {
				logging.Warning("Unable to marshal PUSH_DATA: %v", err)
				continue
			}
			if _, err := g.upstreamConn.Write(buf); err != nil {
				logging.Warning("Error writing PUSH_DATA packet : ", err)
				continue
			}

		case <-time.After(5 * time.Second):
			g.sendPullData()

		case <-shutdownChannel:
			return
		}
	}
}

// Main receiver loop. Similar to the forwarder but grabs PUSH_ACK messages and forwards on a channel.
func (g *SyntheticForwarder) receiver(shutdown chan bool) {
	for {
		select {
		case <-shutdown:
			return
		case <-time.After(4 * time.Second):
			g.sendPullData()
		case p := <-g.incomingMessage:
			select {
			case g.outputChannel <- p:
			case <-time.After(100 * time.Millisecond):
				logging.Warning("Output channel is full! Boo! (waited 100ms to send message)")
			}
		}
	}
}
func (g *SyntheticForwarder) udpReader() {
	buf := make([]byte, 1024)
	for {
		n, _, err := g.downstreamConn.ReadFromUDP(buf)
		if err != nil {
			logging.Warning("Unable to read UDP: %v", err)
			continue
		}

		gwpk := cgw.GwPacket{}
		if err := gwpk.UnmarshalBinary(buf[0:n]); err != nil {
			logging.Warning("Unable to unmarshal gwpkt: %v", err)
			continue
		}
		if gwpk.Identifier != cgw.PullResp {
			continue
		}
		txpk := cgw.TXData{}
		if err := json.Unmarshal([]byte(gwpk.JSONString), &txpk); err != nil {
			logging.Warning("Unable to unmarshal received JSON: %v", err)
			continue
		}
		bytes, err := base64.StdEncoding.DecodeString(txpk.Data.Data)
		if err != nil {
			logging.Warning("Unable to decode hex string: %v", err)
			continue
		}

		g.incomingMessage <- bytes
	}
}

// OutputChannel is messages received from the server
func (g *SyntheticForwarder) OutputChannel() chan []byte {
	return g.outputChannel
}

// Start starts the simulated gateway
func (g *SyntheticForwarder) Start() {
	if !g.initUDP() {
		return
	}
	defer g.upstreamConn.Close()
	defer g.downstreamConn.Close()
	defer close(g.outputChannel)

	forwarderShutdownChannel := make(chan bool)
	receiverShutdownChannel := make(chan bool)

	g.initUDP()
	go g.udpReader()
	go g.forwarder(forwarderShutdownChannel)
	go g.receiver(receiverShutdownChannel)
	for {
		select {
		case <-g.shutdownChannel:
			forwarderShutdownChannel <- true
			receiverShutdownChannel <- true
		}
	}
}
