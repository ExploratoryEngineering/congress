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
	"testing"

	"github.com/ExploratoryEngineering/congress/protocol"
)

func TestBinaryMarshal(t *testing.T) {
	pkt := GwPacket{}
	// A PUSH_DATA sentence with EUI AABBCCDD and the string 'abcdef'
	buffer := []byte{0, 0x11, 0x22, 0, 0xAA, 0xAA, 0xBB, 0xBB, 0xCC, 0xCC, 0xDD, 0xDD, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46}

	err := pkt.UnmarshalBinary(buffer)
	if err != nil {
		t.Fatal("Error unmarshaling bytes for PUSH_DATA: ", err)
	}
	if pkt.GatewayEUI != protocol.EUIFromUint64(0xAAAABBBBCCCCDDDD) {
		t.Fatalf("EUI not what I'd expected: 0x%08X", pkt.GatewayEUI)
	}
	if pkt.Token != 0x1122 {
		t.Fatalf("Token not what I'd expected: 0x%04X", pkt.Token)
	}
	if pkt.JSONString != "ABCDEF" {
		t.Fatalf("String not what I'd expected: %s", pkt.JSONString)
	}

	// PUSH_ACK
	buffer = []byte{0, 0x11, 0x22, 1}
	if pkt.UnmarshalBinary(buffer) != nil {
		t.Fatal("Couldn't unmarshal PUSH_ACK")
	}

	// PULL_DATA
	buffer = []byte{0, 0x11, 0x22, 2, 0xAA, 0xAA, 0xBB, 0xBB, 0xCC, 0xCC, 0xDD, 0xDD}
	if pkt.UnmarshalBinary(buffer) != nil {
		t.Fatal("Couldn't unmarshal PULL_DATA")
	}

	// PULL_RESP
	buffer = []byte{0, 0x11, 0x22, 3, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46}
	if pkt.UnmarshalBinary(buffer) != nil {
		t.Fatal("Couldn't unmarshal PULL_RESP")
	}

	// PULL_ACK
	buffer = []byte{0, 0x11, 0x22, 4}
	if pkt.UnmarshalBinary(buffer) != nil {
		t.Fatal("Couldn't unmarshal PULL_ACK")
	}

	// TX_ACK
	buffer = []byte{0, 0x11, 0x22, 5, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46}
	if pkt.UnmarshalBinary(buffer) != nil {
		t.Fatal("Couldn't unmarshal TX_ACK")
	}

	// Unknown type
	buffer = []byte{0, 0x11, 0x22, 99}
	if pkt.UnmarshalBinary(buffer) == nil {
		t.Fatal("Shouldn't be able to unmarshal unknown types")
	}

	// TODO: Test invalid buffers
	if pkt.UnmarshalBinary([]byte{0}) == nil {
		t.Fatal("Shoulnd't be able to unmarshal tiny buffers")
	}

	if pkt.UnmarshalBinary([]byte{0}) == nil {
		t.Fatal("Shoulnd't be able to unmarshal tiny buffers")
	}

	buffer = []byte{0, 0x11, 0x22, 0}
	if pkt.UnmarshalBinary(buffer) == nil {
		t.Fatal("Shoulnd't be able to unmarshal small PUSH_DATA buffer")
	}

	buffer = []byte{0, 0x11, 0x22, 2}
	if pkt.UnmarshalBinary(buffer) == nil {
		t.Fatal("Shoulnd't be able to unmarshal small PULL_DATA buffer")
	}

}

func TestBinaryUnmarsha(t *testing.T) {
	pkt := GwPacket{
		ProtocolVersion: 0,
		Token:           0x1234,
		Identifier:      PushData,
		GatewayEUI:      protocol.EUIFromUint64(0xAAAABBBBCCCCDDDD),
		JSONString:      "ABCDEF",
	}

	buf, err := pkt.MarshalBinary()
	if err != nil {
		t.Fatal("Got error marshaling struct: ", err)
	}

	if buf == nil {
		t.Fatal("Marshaled buffer is nil")
	}

	pkt = GwPacket{
		Identifier: PullAck,
	}
	_, err = pkt.MarshalBinary()
	if err != nil {
		t.Fatal("Got error marshaling PULL_ACK")
	}

	pkt = GwPacket{
		Identifier: PushAck,
	}
	_, err = pkt.MarshalBinary()
	if err != nil {
		t.Fatal("Got error marshaling PULL_ACK")
	}

	pkt = GwPacket{
		Identifier: PullData,
	}
	_, err = pkt.MarshalBinary()
	if err != nil {
		t.Fatal("Got error marshaling PULL_DATA")
	}

	pkt = GwPacket{
		Identifier: PullResp,
	}
	_, err = pkt.MarshalBinary()
	if err != nil {
		t.Fatal("Got error marshaling PULL_RESP")
	}

	pkt = GwPacket{
		Identifier: PullResp,
	}
	_, err = pkt.MarshalBinary()
	if err != nil {
		t.Fatal("Got error marshaling PULL_RESP")
	}

	pkt = GwPacket{
		Identifier: TxAck,
	}
	_, err = pkt.MarshalBinary()
	if err != nil {
		t.Fatal("Got error marshaling PULL_DATA")
	}

	pkt = GwPacket{
		Identifier: UnknownType,
	}
	_, err = pkt.MarshalBinary()
	if err == nil {
		t.Fatal("Expected error marshaling unknown type")
	}

}

func TestBinaryMarshalUnmarshal(t *testing.T) {
	pkt := GwPacket{
		ProtocolVersion: 1,
		Token:           0xAABB,
		Identifier:      PullResp,
		JSONString: `{"txpk":{"imme":false,"tmst":254014692,"freq":868.5,"rfch":0,"modu":"LORA","datr":"SF12BW125","size":17,"data":"oGL34y/vUiWG
+OYcwPZAKgA"}}`,
	}

	bytes, err := pkt.MarshalBinary()
	if err != nil {
		t.Fatalf("Couldn't marshal packet (%v): %v ", pkt, err)
	}

	pk2 := GwPacket{}
	if err := pk2.UnmarshalBinary(bytes); err != nil {
		t.Fatalf("Got error unmarshaling bytes (source=%v, bytes=%v):  %v", pkt, bytes, err)
	}

	if pkt != pk2 {
		t.Fatalf("Not the same packet (original: %v != copy: %v)", pkt, pk2)
	}
}
