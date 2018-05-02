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
	"encoding/binary"
	"testing"
)

func TestAppNonceGenerator(t *testing.T) {
	joinAccept := JoinAcceptPayload{}
	joinAccept.AppNonce, _ = joinAccept.GenerateAppNonce()

	if joinAccept.AppNonce[0] == 0 && joinAccept.AppNonce[1] == 0 && joinAccept.AppNonce[2] == 0 {
		t.Fatal("AppNonce is 0,0,0. Did not expect that. (but it might happen once in a blue moon)")
	}

}

func TestJoinAcceptEncode(t *testing.T) {
	var err error
	joinAccept := JoinAcceptPayload{}
	buffer := make([]byte, 16)
	joinAccept.AppNonce = [3]byte{0xAB, 0xCD, 0xEF}
	joinAccept.NetID = 0x00AABBCC
	joinAccept.DevAddr, err = DevAddrFromString("1FFFFFFF")
	if err != nil {
		t.Fatal("Could not encode DevAddr: ", err)
	}
	joinAccept.RxDelay = 0x99
	joinAccept.DLSettings = DLSettings{RX1DRoffset: 2, RX2DataRate: 3}
	joinAccept.CFList = CFList{}

	pos := 0
	if err := joinAccept.encode(buffer, &pos); err != nil {
		t.Fatal("Could not encode JoinAccept into buffer: ", err)
	}
	if pos != 12 {
		t.Fatalf("Position not updated (pos = %d)", pos)
	}

	// AppNonce should be the first three bytes
	if buffer[0] != 0xAB || buffer[1] != 0xCD || buffer[2] != 0xEF {
		t.Fatalf("buffer[0:3] not ABCDEF (is %v)", buffer[0:3])
	}
	// NetID the next three
	if buffer[3] != 0xAA || buffer[4] != 0xBB || buffer[5] != 0xCC {
		t.Fatalf("buffer[3:6] not AABBCC (is %v)", buffer[3:6])
	}
	// DevAddr next four bytes
	devAddr := binary.LittleEndian.Uint32(buffer[6:])
	if devAddr != joinAccept.DevAddr.ToUint32() {
		t.Fatalf("buffer[6:10] not DevAddr (is %08x expected %08x)", devAddr, joinAccept.DevAddr.ToUint32())
	}
	// ...and DLSettings. Everything is encoded into a single byte so we should have 0x23
	if buffer[10] != 0x23 {
		t.Fatalf("buffer[10] not 0x23 (is %02x)", buffer[10])
	}

	// Finally, RxDelay is set
	if buffer[11] != 0x99 {
		t.Fatalf("RxDelay isn't set")
	}

	// Repeat encoding. Should give error since there's no room
	if err := joinAccept.encode(buffer, &pos); err == nil {
		t.Fatal("Expected error when encoding with too small buffer.")
	}
}

func TestJoinAcceptEncodeDecode(t *testing.T) {
	j1 := JoinAcceptPayload{
		AppNonce:   [3]byte{1, 2, 3},
		NetID:      0x0a0b0c,
		DevAddr:    DevAddrFromUint32(0x04030201),
		DLSettings: DLSettings{1, 2},
		RxDelay:    99,
		CFList:     CFList{},
	}
	j2 := JoinAcceptPayload{}

	buf := make([]byte, 1024)
	pos := 0
	if err := j1.encode(buf, &pos); err != nil {
		t.Fatalf("Got error encoding JoinAccept: %v", err)
	}
	pos = 0
	if err := j2.decode(buf, &pos); err != nil {
		t.Fatalf("Got error decoding JoinAccept: %v", err)
	}
	if j1 != j2 {
		t.Fatalf("j1 and j2 aren't the same: %+v != %+v", j1, j2)
	}
}

func TestBasicEncodeDecodeJoinAccept(t *testing.T) {
	basicDecoderTests(t, &JoinAcceptPayload{})
	basicEncoderTests(t, &JoinAcceptPayload{})
}
