package protocol

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
	"encoding/binary"
	"testing"
)

// TestVerifyMIC calculates MIC (Message Integrity Code) from message and compares the calculated value with transmitted value
func TestVerifyMIC(t *testing.T) {
	appSKey, _ := AESKeyFromString("E001 2A22 25B8 585E DCEC 7042 4798 C510")
	nwkSKey, _ := AESKeyFromString("3C5E 5C9F 469E EF3E 02CC D4FF 9531 31BA")
	messageStruct := createUnencryptedTestMessage()

	messageStruct.encrypt(nwkSKey, appSKey)

	if messageStruct.MIC != 0x22CBE65F {
		t.Error("MIC verification failed. Unexpected MIC:", messageStruct.MIC)
	}
}

// Mic should be 0xB08D7C07 (Big endian)
var joinRequestBytes = []string{
	"AAgHBgUEAwIBvrrvvr66774RXbCNfAc=",
	"AAgHBgUEAwIBvrrvvr66776KPsOMMuc=",
	"AAgHBgUEAwIBvrrvvr66775Wreckkbc=",
	"AAgHBgUEAwIBvrrvvr66775+xPtJBQI="}

func TestVerifyJoinRequestMIC(t *testing.T) {
	for _, v := range joinRequestBytes {

		buffer, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			t.Fatal("Couldn't decode string: ", err)
		}

		appKey, _ := AESKeyFromString("0102030405060708 0102030405060708")
		messageStruct := NewPHYPayload(ConfirmedDataUp)
		pos := 0
		if err := messageStruct.MHDR.decode(buffer, &pos); err != nil {
			t.Fatal("Could not decode bytes: ", err)
		}
		if messageStruct.MHDR.MType != JoinRequest {
			t.Fatal("Not a JoinRequest message. MType is ", messageStruct.MHDR)
		}

		appEUI := binary.LittleEndian.Uint64(buffer[1:])
		devEUI := binary.LittleEndian.Uint64(buffer[9:])

		payload := make([]byte, 19)
		payload[0] = buffer[0] // MHDR byte, mtype = 0, version = 0 => everything 0
		binary.LittleEndian.PutUint64(payload[1:], appEUI)
		binary.LittleEndian.PutUint64(payload[9:], devEUI)
		pos = 17
		payload[pos] = buffer[17]
		payload[pos+1] = buffer[18]

		mic, err := messageStruct.CalculateJoinRequestMIC(appKey, payload)
		if err != nil {
			t.Fatal("Couldn't calculate MIC: ", err)
		}

		existingMIC := binary.LittleEndian.Uint32(buffer[19:])

		if mic != existingMIC {
			t.Fatalf("Calculated MIC does not match existing MIC. Got 0x%08x, expected 0x%08x", mic, existingMIC)
		}
	}
}

func TestJoinAcceptMIC(t *testing.T) {
	payload := make([]byte, 16)
	joinAccept := JoinAcceptPayload{
		AppNonce:   [3]byte{1, 2, 3},
		NetID:      0x00123456,
		DevAddr:    DevAddr{NwkID: 1, NwkAddr: 2},
		DLSettings: DLSettings{RX1DRoffset: 1, RX2DataRate: 1},
		RxDelay:    1,
		CFList:     CFList{},
	}
	pos := 0
	if err := joinAccept.encode(payload, &pos); err != nil {
		t.Fatal("Got error encoding data: ", err)
	}

	appKey, _ := AESKeyFromString("0102030405060708 0102030405060708")
	p := NewPHYPayload(ConfirmedDataUp)
	mic, err := p.CalculateJoinAcceptMIC(appKey, payload)
	if err != nil {
		t.Fatal("Got error calculating MIC: ", err)
	}
	if mic != 0xf2511d15 {
		t.Fatalf("Unexpected MIC. Got 0x%08x, expected 0x9d9dddd2", mic)
	}
}
