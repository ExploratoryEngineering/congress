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
	"encoding/binary"
	"fmt"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/logging"
)

// The protocol parts - ie the Semtech packet forwarder

// Message identifiers for the packet forwarder protocol
const (
	PushData    = 0  // PushData (aka PUSH_DATA) is used by the gateway to push data to the server
	PushAck     = 1  // PushAck (aka PUSH_ACK) is sent from the server to the gateway as a response to PushData packets
	PullData    = 2  // PullData (aka PULL_DATA) is used to poll from the gateway to the server. It is sent periodically
	PullResp    = 3  // PullResp (aka PULL_RESP) is sent from the server to the gateway indicating that it is connected to the network
	PullAck     = 4  // PullAck (aka PULL_ACK) is sent from the gateway to the server to signal that the network is available
	TxAck       = 5  // TxAck (aka TX_ACK) is sent from the gateway to the server to acknowledge packets
	UnknownType = 99 // UnknownType represents an unknown type identifier

)

// GwPacket represents a packet received from (or sent to) the gateway
type GwPacket struct {
	ProtocolVersion uint8        // ProtocolVersion protocol version reported to/from the gateway
	Token           uint16       // Token is sent by the gateway and re-used by the server when sending responses.
	Identifier      int          // Identifier is the packet identifier (PUSH_DATA/PUSH_ACK etc)
	GatewayEUI      protocol.EUI // GatewayEUI holds the gateway's EUI. It might be 0 if it isn't set in the originating packet
	JSONString      string       // Option JSON sentence(s) sent by or to the gateway
	Host            string       // The IP address of the gateway that sent the message
	Port            int          // The port of the gateway that sent the message
}

// UnmarshalBinary decodes a byte buffer into a GwPacket structure
func (pkt *GwPacket) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("buffer too short. Needs 4 bytes, buffer is %d", len(data))
	}
	pkt.ProtocolVersion = data[0]
	pkt.Token = uint16(data[1]) << 8
	pkt.Token += uint16(data[2])
	switch data[3] {
	case 0:
		pkt.Identifier = PushData
		if len(data) < 12 {
			return fmt.Errorf("buffer too short. Needs 12 bytes, buffer is %d", len(data))
		}
		val := binary.BigEndian.Uint64(data[4:12])
		pkt.GatewayEUI = protocol.EUIFromUint64(val)
		if len(data) > 12 {
			pkt.JSONString = string(data[12:])
		}

	case 1:
		pkt.Identifier = PushAck

	case 2:
		pkt.Identifier = PullData
		if len(data) < 12 {
			return fmt.Errorf("buffer too short. Needs 12 bytes, buffer is %d", len(data))
		}
		val := binary.BigEndian.Uint64(data[4:12])
		pkt.GatewayEUI = protocol.EUIFromUint64(val)

	case 3:
		pkt.Identifier = PullResp
		if len(data) > 4 {
			pkt.JSONString = string(data[4:])
		}

	case 4:
		pkt.Identifier = PullAck

	case 5:
		pkt.Identifier = TxAck
		if len(data) > 4 {
			pkt.JSONString = string(data[4:])
		}

	default:
		pkt.Identifier = UnknownType
		return fmt.Errorf("unknown packet identifier: %d", data[3])
	}
	return nil
}

// MarshalBinary marshals the packet into a binary byte buffer
func (pkt *GwPacket) MarshalBinary() ([]byte, error) {
	// Max length is 12 + length of string (for PushData)
	data := make([]byte, 12+len(pkt.JSONString))
	data[0] = pkt.ProtocolVersion
	data[1] = byte(pkt.Token >> 8 & 0xFF)
	data[2] = byte(pkt.Token & 0xFF)

	switch pkt.Identifier {
	case PullAck:
		data[3] = PullAck
		return data[:4], nil

	case PushAck:
		data[3] = PushAck
		return data[:4], nil

	case PullData:
		data[3] = PullData
		copy(data[4:], pkt.GatewayEUI.Octets[:])
		return data[:12], nil

	case PushData:
		data[3] = PushData
		copy(data[4:], pkt.GatewayEUI.Octets[:])
		copy(data[12:], pkt.JSONString)
		packetLen := 12 + len(pkt.JSONString)
		return data[:packetLen], nil

	case PullResp:
		data[3] = PullResp
		copy(data[4:], pkt.JSONString)
		packetLen := 4 + len(pkt.JSONString)
		return data[:packetLen], nil

	case TxAck:
		data[3] = TxAck
		copy(data[4:], pkt.JSONString)
		packetLen := 4 + len(pkt.JSONString)
		return data[:packetLen], nil

	default:
		logging.Warning("Unknown packet identifier %d\n", pkt.Identifier)
		return nil, fmt.Errorf("don't know how to encode packet identifier %d", pkt.Identifier)
	}
}
