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
	"time"

	"github.com/ExploratoryEngineering/congress/monitoring"

	"sync"

	"github.com/ExploratoryEngineering/congress/band"
	"github.com/ExploratoryEngineering/congress/events/gwevents"
	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

var defaultBand band.FrequencyPlan

func init() {
	var err error
	defaultBand, err = band.NewBand(band.EU868Band)
	if err != nil {
		logging.Error("Unable to create EU868 band instance: %v", err)
	}
}

// GenericPacketForwarder is the generic packet forwarder provided by
// Semtech. It has its weak points but it is the smallest common
// denominator for all gateways on the market.
type GenericPacketForwarder struct {
	input        chan server.GatewayPacket // Input to the gateway, ie data that should be sent to the gateway
	output       chan server.GatewayPacket // Output from the gateway; ie data received from the gateway
	udpInput     chan GwPacket             // Internal channel for packets that should be sent on the UDP interface
	udpOutput    chan GwPacket             // Internal channel for packets that are received on the UDP interface
	serverPort   int                       // Server port to listen on
	terminate    chan bool
	storage      storage.GatewayStorage
	context      *server.Context
	mutex        *sync.Mutex    // Mutex for pullAckPort map
	pullAckPorts map[string]int // Map of port <-> gateway
}

// Start launches the generic packet forwarder. It does not return until the
// gateway shuts down.
func (p *GenericPacketForwarder) Start() {
	// Set up server port (the one the gateway is going to connect to)
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", p.serverPort))
	if err != nil {
		logging.Error("Unable to create UDP socket: ", err)
		return
	}

	logging.Info("Generic Packet Forwarder listening on port %d", p.serverPort)
	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		logging.Error("Unable to listen on UDP port %d: %v", p.serverPort, err)
		return
	}

	go p.udpSender(serverConn)
	go p.udpReader(serverConn)
	p.mainLoop(serverConn)
}

// Stop stops the packet forwarder and closes the channels
func (p *GenericPacketForwarder) Stop() {
	close(p.input)
}

// Output returns the output channel for the gateway. A message will be sent
// on this channel every time the gateway has sent a message to the server.
func (p *GenericPacketForwarder) Output() <-chan server.GatewayPacket {
	return p.output
}

// Input returns the input channel for the gateway. When a message is received
// on this channel the message will be forwarded to the gateway.
func (p *GenericPacketForwarder) Input() chan<- server.GatewayPacket {
	return p.input
}

// NewGenericPacketForwarder creates a new generic packet forwarder listening on
// a port. There's no authentication (which is TBD) or validation of the gateway
// (which is a bad idea). The serverPort parameter specifies the port the server
// will listen on and the gatewayPort specifies which port the gateway is
// supposed to listen on. There's no need to configure the gateways since the
// gateway's IP will be attached to the received data.
func NewGenericPacketForwarder(serverPort int, storage storage.GatewayStorage, context *server.Context) *GenericPacketForwarder {
	return &GenericPacketForwarder{
		input:        make(chan server.GatewayPacket),
		output:       make(chan server.GatewayPacket),
		serverPort:   serverPort,
		udpInput:     make(chan GwPacket),
		udpOutput:    make(chan GwPacket),
		terminate:    make(chan bool),
		storage:      storage,
		context:      context,
		mutex:        &sync.Mutex{},
		pullAckPorts: make(map[string]int),
	}
}

// Start reading from UDP and forward to the main loop. The main loop listens
// for the UDP and external input at the same time.
func (p *GenericPacketForwarder) udpReader(serverConn *net.UDPConn) {
	buf := make([]byte, 8192)
	for {
		select {
		case <-p.terminate:
			logging.Debug("Terminate signal received. Closing UDP reader")
			return
		default:
			// Nothing
		}
		n, addr, err := serverConn.ReadFromUDP(buf)
		if err != nil {
			logging.Warning("Unable to read from UDP socket at %v: %v", serverConn.RemoteAddr(), err)
			<-time.After(1000 * time.Millisecond)
			continue
		}
		var pkt GwPacket
		err = pkt.UnmarshalBinary(buf[0:n])
		pkt.Host = addr.IP.String()
		pkt.Port = addr.Port

		if err != nil {
			logging.Warning("Unable to unmarshal buffer received from %v: %v",
				serverConn.RemoteAddr(), err)
			continue
		}

		p.udpInput <- pkt
	}
}

func (p *GenericPacketForwarder) setPullAckPort(eui protocol.EUI, port int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.pullAckPorts[eui.String()] = port
}

func (p *GenericPacketForwarder) getPullAckPort(eui protocol.EUI) int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	port, exists := p.pullAckPorts[eui.String()]
	if !exists {
		logging.Warning("Gateway with EUI %s haven't sent a PULL_DATA yet so we don't know the port", eui)
	}
	return port
}

func (p *GenericPacketForwarder) udpSender(serverConn *net.UDPConn) {
	defer serverConn.Close()
	for val := range p.udpOutput {
		buffer, err := val.MarshalBinary()
		if err != nil {
			logging.Error("Unable to marshal packet forwarder data: %v", err)
			continue
		}
		if val.JSONString != "" {
			p.context.GwEventRouter.Publish(val.GatewayEUI, gwevents.NewTx(val.JSONString))
		}
		targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", val.Host, val.Port))
		if err != nil {
			logging.Warning("Unable to resolve target address for gateway (%s:%d): %v", val.Host, val.Port, err)
			continue
		}
		_, _, err = serverConn.WriteMsgUDP(buffer, nil, targetAddr)
		if err != nil {
			logging.Warning("Unable to write UDP message to gateway at %s: %v", targetAddr, err)
			continue
		}
		monitoring.GatewayOut.Increment()
	}
	logging.Debug("UDP output channel closed. Terminating UDP sender")
}

// Wait for either external or internal input before sending. If there's internal
// input a protocol response is required. if it is an external input there's
// data to be sent. The main loop handles the protocol itself and translates
// between the wire format and the external representation.
func (p *GenericPacketForwarder) mainLoop(serverConn *net.UDPConn) {
	defer serverConn.Close()
	for {
		select {
		case val, ok := <-p.input:
			val.SectionTimer.Begin(monitoring.TimeGatewaySend)
			// Generate a txpk message, aka PULL_RESP
			if !ok {
				logging.Debug("Input channel for forwarder closed. Terminating")
				// Close all channels, connections and terminate

				close(p.udpInput)
				close(p.udpOutput)
				close(p.output)
				p.terminate <- true
				return
			}
			p.encodeAndSend(val)
			val.OutTimer.End()
			val.SectionTimer.End()

		case val := <-p.udpInput:
			// This is a message from the server. If it is a JSON sentence decoded it and forward i
			switch val.Identifier {

			case PullData:
				// Send PullAck with same version and token
				logging.Debug("PULL_DATA received from %s, sending PULL_ACK response", val.GatewayEUI)
				p.setPullAckPort(val.GatewayEUI, val.Port)
				p.udpOutput <- GwPacket{
					GatewayEUI:      val.GatewayEUI,
					Identifier:      PullAck,
					Token:           val.Token,
					Host:            val.Host,
					Port:            val.Port,
					ProtocolVersion: val.ProtocolVersion,
				}
				p.context.GwEventRouter.Publish(val.GatewayEUI, gwevents.NewKeepAlive())

			case PushData:
				logging.Debug("PUSH_DATA received from %s: %s", val.GatewayEUI, val.JSONString)
				if !p.context.Config.DisableGatewayChecks {
					gw, err := p.storage.Get(val.GatewayEUI, model.SystemUserID)
					if err != nil {
						logging.Info("Unable to locate gateway with EUI %s: %v", val.GatewayEUI, err)
						continue
					}
					if gw.StrictIP && gw.IP.String() != val.Host {
						logging.Warning("IP mismatch for gateway with EUI %s: %s (should be %s)", val.GatewayEUI, gw.IP, val.Host)
						continue
					}
				}
				p.context.GwEventRouter.Publish(val.GatewayEUI, gwevents.NewRx(val.JSONString))

				// Send PushAck with same version and token
				p.decodeReceivedJSON(val)
				p.udpOutput <- GwPacket{
					Identifier:      PushAck,
					Token:           val.Token,
					Host:            val.Host,
					Port:            val.Port,
					GatewayEUI:      val.GatewayEUI,
					ProtocolVersion: val.ProtocolVersion,
				}
				monitoring.GatewayIn.Increment()
			case TxAck:
				// Ignore (for now)
			default:
				logging.Info("Don't know how to handle input with identifier=%d from gateway", val.Identifier)
			}
		}
	}
}

// This is the default setup for the Semtech packet forwarder/EU868 band config.
func (p *GenericPacketForwarder) lookupFrequency(rfchain uint8, channel uint8) float32 {
	switch channel {
	case 0:
		return 868.1
	case 1:
		return 868.3
	case 2:
		return 868.5
	case 3:
		return 867.1
	case 4:
		return 867.3
	case 5:
		return 867.5
	case 6:
		return 867.7
	case 7:
		return 867.9
	}

	logging.Warning("Unknown channel: %d. Returning 868.1MHz", channel)
	return 868.1

}

// Unmarshal and forward JSON from gateway
func (p *GenericPacketForwarder) decodeReceivedJSON(val GwPacket) {
	rxData := RXData{}

	incomingTimer := monitoring.NewTimer()
	incomingTimer.Begin(monitoring.TimeIncoming)

	timer := monitoring.NewTimer()
	timer.Begin(monitoring.TimeGatewayReceive)
	var err error
	if err = json.Unmarshal([]byte(val.JSONString), &rxData); err != nil {
		logging.Info("Unable to unmarshal JSON from %s:%d: %v (json=%s)", val.Host, val.Port, err, val.JSONString)
		return
	}

	for _, packet := range rxData.Data {
		gwPacket := server.GatewayPacket{
			Radio: server.RadioContext{
				Frequency: p.lookupFrequency(packet.ConcentratorRFChain, packet.ConcentratorChannel),
				DataRate:  packet.DataRateID,
				Channel:   packet.ConcentratorChannel,
				RFChain:   packet.ConcentratorRFChain,
				Band:      defaultBand,
				RX1Delay:  0,
				RX2Delay:  0,
				RSSI:      packet.RSSI,
				SNR:       packet.LoraSNRRatio,
			},
			Gateway: server.GatewayContext{
				GatewayEUI:      val.GatewayEUI,
				GatewayHost:     val.Host,
				GatewayPort:     val.Port,
				GatewayClock:    packet.Timestamp,
				ProtocolVersion: val.ProtocolVersion,
			},
			ReceivedAt:   time.Now(),
			SectionTimer: timer,
			InTimer:      incomingTimer,
		}
		if gwPacket.RawMessage, err = base64.StdEncoding.DecodeString(packet.RFPackets); err != nil {
			logging.Info("Unable to convert base64 string into bytes: %v (source=%s)", err, packet.RFPackets)
			return
		}
		gwPacket.SectionTimer.End()
		monitoring.Stopwatch(monitoring.GatewayChannelOut, func() {
			p.output <- gwPacket
		})
	}
}

// Encode and send data as JSON to gateway
func (p *GenericPacketForwarder) encodeAndSend(packet server.GatewayPacket) {
	// Create a PULL_RESP packet for the gateway
	// Timestamp is in us; use precomputed RXDelay value
	timestamp := packet.Gateway.GatewayClock + 1000000*uint32(packet.Radio.RX1Delay)
	outputPkt := Txpk{
		Timestamp:    timestamp,              // us clock
		Frequency:    packet.Radio.Frequency, // packet.TransmitFrequency,
		RFChain:      0,                      // This is the default for the packet forwarder
		Data:         base64.StdEncoding.EncodeToString(packet.RawMessage),
		Modulation:   "LORA",
		EccCoding:    "4/5",
		LoraInvPol:   true,
		PayloadSize:  len(packet.RawMessage),
		LoRaDataRate: packet.Radio.DataRate,
	}
	outputStruct := TXData{Data: outputPkt}

	buffer, err := json.Marshal(outputStruct)
	if err != nil {
		logging.Info("Unable to marshal JSON for txpk: %v", err)
		return
	}
	p.udpOutput <- GwPacket{
		Identifier:      PullResp,
		Token:           uint16(rand.Int() & 0xFFFF), // This is unused in v1
		Host:            packet.Gateway.GatewayHost,
		Port:            p.getPullAckPort(packet.Gateway.GatewayEUI),
		ProtocolVersion: packet.Gateway.ProtocolVersion,
		GatewayEUI:      packet.Gateway.GatewayEUI,
		JSONString:      string(buffer),
	}
	timeToProcess := time.Now().Sub(packet.ReceivedAt)
	const assumedLatency = 0.2
	// Assume 100ms latency between gateway and
	// Congress. This is rougly what we can expect in Europe. Norway -> Ireland
	// is about 50 ms; further south is is easily 100ms (or more).
	if timeToProcess.Seconds() > (packet.Deadline - assumedLatency) {
		logging.Error("Packet to %s missed deadline of %.2f seconds with assumedLatency of %.2f (took %.2f s)",
			packet.Gateway.GatewayEUI, packet.Deadline, assumedLatency, timeToProcess.Seconds())
		monitoring.MissedDeadline.Increment()
	}
}
