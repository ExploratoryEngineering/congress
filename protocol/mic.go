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

	"github.com/ExploratoryEngineering/congress/cmac"
)

// CalculateMIC calculates the Message Integrity Code [4.4]
func (p *PHYPayload) CalculateMIC(nwkSKey AESKey, message []byte) (uint32, error) {
	b0 := make([]byte, 16)
	b0[0] = 0x49
	b0[1] = 0
	b0[2] = 0
	b0[3] = 0
	b0[4] = 0
	if p.MHDR.MType.Uplink() {
		b0[5] = 0
	} else {
		b0[5] = 1
	}
	binary.LittleEndian.PutUint32(b0[6:], p.MACPayload.FHDR.DevAddr.ToUint32())
	binary.LittleEndian.PutUint32(b0[10:], uint32(p.MACPayload.FHDR.FCnt))
	b0[14] = 0
	b0[15] = byte(len(message))

	fullMessage := append(b0, message[0:]...)

	return p.calculateMICFromBuffer(nwkSKey, fullMessage)
}

// calculateMICFromBuffer calculates a MIC using the given key and buffer.
func (p *PHYPayload) calculateMICFromBuffer(key AESKey, payload []byte) (uint32, error) {
	cmac, err := cmac.AESCMAC(key.Key[0:], payload)
	if err != nil {
		return 0, err
	}
	mic := binary.LittleEndian.Uint32(cmac[0:4])

	return mic, nil

}

// CalculateJoinAcceptMIC calculates the JoinAccept MIC. The payload is expected to be
// the payload that is sent to the end-device (6.2.5):
//    MHDR | AppNonce | NetID | DevAddr | DLSettings | RxDelay | CFList
// The MHDR field is used to build the buffer used when calculating the MIC.
func (p *PHYPayload) CalculateJoinAcceptMIC(appKey AESKey, payload []byte) (uint32, error) {
	return p.calculateMICFromBuffer(appKey, payload)
}

// CalculateJoinRequestMIC calculates the JoinRequest MIC. The payload is the same payload as
// the end-device sends to the network server (6.2.4):
//     AppEUI | DevEUI | DevNonce
// The MHDR field is used to build the buffer used when calculating the MIC.
func (p *PHYPayload) CalculateJoinRequestMIC(appKey AESKey, payload []byte) (uint32, error) {
	return p.calculateMICFromBuffer(appKey, payload)
}
