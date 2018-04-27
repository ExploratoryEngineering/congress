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
	"reflect"
	"testing"

	"github.com/ExploratoryEngineering/logging"
)

func TestDecodeJoinAccept(t *testing.T) {
	aesKey, _ := AESKeyFromString("01020304 05060708 01020304 05060708")
	input := PHYPayload{
		MHDR: MHDR{MType: JoinAccept, MajorVersion: MaxSupportedVersion},
		JoinAcceptPayload: JoinAcceptPayload{
			AppNonce: [3]byte{0, 1, 2},
			NetID:    0x00010203,
			DevAddr:  DevAddr{NwkID: 1, NwkAddr: 2},
			DLSettings: DLSettings{
				RX1DRoffset: 1,
				RX2DataRate: 2,
			},
			RxDelay: 4,
			CFList:  CFList{},
		},
	}
	buffer, err := input.EncodeJoinAccept(aesKey)
	if err != nil {
		t.Fatal("Got error encoding JoinAccept: ", err)
	}

	payload := PHYPayload{}

	err = payload.UnmarshalBinary(buffer)
	if err != nil {
		t.Fatal("Got error decoding JoinAccept payload: ", err)
	}

	if payload.MHDR.MType != JoinAccept {
		t.Fatalf("Didn't get a JoinAccept message: %v", payload)
	}

	if err = payload.DecodeJoinAccept(aesKey, buffer); err != nil {
		t.Fatal("Got error decoding JoinAccept payload: ", err)
	}
	if payload.MIC != input.MIC {
		t.Fatalf("MIC doesn't match. Expected %v but got %v", input.MIC, payload.MIC)
	}
	if payload.JoinAcceptPayload != input.JoinAcceptPayload {
		t.Fatalf("Not the same output as input in: %v out: %v", input.JoinAcceptPayload, payload.JoinAcceptPayload)
	}
}

func TestDecodeShortBuffers(t *testing.T) {
	p := NewPHYPayload(UnconfirmedDataUp)

	// Ensure byte buffers less than 12 bytes (MHDR + FHDR + FCnt + MIC) are rejected
	buf, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("Got error marshaling test binary: %v", err)
	}
	for i := 0; i < 12; i++ {
		if err := p.UnmarshalBinary(buf[0:i][:]); err == nil {
			t.Fatal("Short buffers should be rejected but error was nil")
		}
	}
}

func createEncryptedTestMessage() PHYPayload {
	message := PHYPayload{
		MHDR: MHDR{
			MType:        UnconfirmedDataUp,
			MajorVersion: LoRaWANR1,
		},
		MACPayload: MACPayload{
			FHDR: FHDR{
				DevAddr: DevAddr{
					NwkID:   0,
					NwkAddr: 0x1E672E6,
				},
				FCtrl: FCtrl{
					ADR:       true,
					ADRACKReq: true,
					ACK:       false,
					FPending:  false,
					ClassB:    false,
					FOptsLen:  0,
				},
				FCnt:  24,
				FOpts: NewFOptsSet(UnconfirmedDataUp),
			},
			FPort:      12,
			FRMPayload: []byte{64, 238, 230, 130, 88, 184, 42, 7, 126, 23, 44, 234, 243, 24, 7, 221, 192, 181, 108, 89, 132, 44, 165, 42, 244},
		},
		MIC: 0x22CBE65F,
	}
	return message
}

func createUnencryptedTestMessage() PHYPayload {
	message := PHYPayload{
		MHDR: MHDR{
			MType:        UnconfirmedDataUp,
			MajorVersion: LoRaWANR1,
		},
		MACPayload: MACPayload{
			FHDR: FHDR{
				DevAddr: DevAddr{
					NwkID:   0,
					NwkAddr: 0x1E672E6,
				},
				FCtrl: FCtrl{
					ADR:       true,
					ADRACKReq: true,
					ACK:       false,
					FPending:  false,
					ClassB:    false,
					FOptsLen:  0,
				},
				FCnt:  24,
				FOpts: NewFOptsSet(UnconfirmedDataUp),
			},
			FPort:      12,
			FRMPayload: []byte{77, 114, 76, 105, 118, 105, 110, 103, 115, 116, 111, 110, 101, 32, 73, 32, 80, 114, 101, 115, 117, 109, 101, 255, 25},
			// ASCII : 	       M   r    L   i    v    i    n    g    s    t    o    n    e        I       P   r    e    s    u    m    e    �
		},
		MIC: 0,
	}
	return message
}

func createTestMessageWithoutPayload() PHYPayload {
	message := PHYPayload{
		MHDR: MHDR{
			MType:        UnconfirmedDataUp,
			MajorVersion: LoRaWANR1,
		},
		MACPayload: MACPayload{
			FHDR: FHDR{
				DevAddr: DevAddr{
					NwkID:   0,
					NwkAddr: 0x1E672E6,
				},
				FCtrl: FCtrl{
					ADR:       true,
					ADRACKReq: true,
					ACK:       false,
					FPending:  false,
					ClassB:    false,
					FOptsLen:  0,
				},
				FCnt:  24,
				FOpts: NewFOptsSet(UnconfirmedDataUp),
			},
			FPort:      12,
			FRMPayload: []byte{},
		},
		MIC: 0,
	}
	return message
}

func TestDecryption(t *testing.T) {
	appSKey, _ := AESKeyFromString("E001 2A22 25B8 585E DCEC 7042 4798 C510")
	nwkSKey, _ := AESKeyFromString("3C5E 5C9F 469E EF3E 02CC D4FF 9531 31BA")

	messageStruct := createEncryptedTestMessage()

	messageStruct.Decrypt(nwkSKey, appSKey)
	plaintextActual := messageStruct.MACPayload.FRMPayload
	plaintextExpected := []byte{77, 114, 76, 105, 118, 105, 110, 103, 115, 116, 111, 110, 101, 32, 73, 32, 80, 114, 101, 115, 117, 109, 101, 255, 25}
	// ASCII Translation: 		M   r    L   i    v    i    n    g    s    t    o    n    e        I       P   r    e    s    u    m    e    �

	if !reflect.DeepEqual(plaintextActual, plaintextExpected) {
		t.Error("Decryption failed.")
	}
}

func TestEncryption(t *testing.T) {
	appSKey, _ := AESKeyFromString("E001 2A22 25B8 585E DCEC 7042 4798 C510")
	nwkSKey, _ := AESKeyFromString("3C5E 5C9F 469E EF3E 02CC D4FF 9531 31BA")

	// Insert an unecrypted into FRMPayload and encrypt it in place
	messageStructActual := createTestMessageWithoutPayload()
	plainTextPayload := []byte{77, 114, 76, 105, 118, 105, 110, 103, 115, 116, 111, 110, 101, 32, 73, 32, 80, 114, 101, 115, 117, 109, 101, 255, 25}
	// ASCII Translation: 	  M   r    L   i    v    i    n    g    s    t    o    n    e        I       P   r    e    s    u    m    e    �
	messageStructActual.MACPayload.FRMPayload = plainTextPayload
	messageStructActual.encrypt(nwkSKey, appSKey)

	messageStructExpected := createEncryptedTestMessage()

	if !reflect.DeepEqual(messageStructActual.MACPayload.FRMPayload, messageStructExpected.MACPayload.FRMPayload) {
		t.Error("Decryption failed.")
	}
}

func TestDecrypt_PleasedToMeetYouMrStanley(t *testing.T) {
	// First successfully decrypted and verified message on LoRa device
	// MIC Verification and decryption is not done in place.
	data := "YAQDAgGAswAHQkb9gVSf4wOzPZzfUjxwwQgG4wLfaGBzZ22I0L1VeNRUy0VT"
	sDec, _ := base64.StdEncoding.DecodeString(data)

	payload := PHYPayload{}
	err := payload.UnmarshalBinary(sDec)
	if err != nil {
		t.Fatal(err)
	}
	var ExpectedNetworkID uint8
	var ExpectedNetworkAddress uint32 = 0x1020304
	var ExpectedSequenceNumber uint16 = 0xB3
	var ExpectedMIC uint32 = 0x5345CB54

	var ExpectedPlaintextPayload = "Pleased to meet you Mr Stanley!!"
	assertEqual(t, payload.MACPayload.FHDR.DevAddr.NwkID, ExpectedNetworkID, "Incorrect network ID.")
	assertEqual(t, payload.MACPayload.FHDR.DevAddr.NwkAddr, ExpectedNetworkAddress, "Incorrect network address.")
	assertEqual(t, payload.MACPayload.FHDR.FCnt, ExpectedSequenceNumber, "Incorrect sequence number.")
	assertEqual(t, payload.MIC, ExpectedMIC, "Incorrect MIC")

	appSKey, _ := AESKeyFromString("0102 0304 0506 0708 090A 0B0C 0D0E 0F10")
	nwkSKey, _ := AESKeyFromString("0102 0304 0506 0708 090A 0B0C 0D0E 0F10")
	payload.Decrypt(nwkSKey, appSKey)

	assertEqual(t, string(payload.MACPayload.FRMPayload), ExpectedPlaintextPayload, "Incorrect payload.")
}

func createPleasedToMeetYouMrStanleyMessage() PHYPayload {
	message := PHYPayload{
		MHDR: MHDR{
			MType:        UnconfirmedDataDown,
			MajorVersion: LoRaWANR1,
		},
		MACPayload: MACPayload{
			FHDR: FHDR{
				DevAddr: DevAddr{
					NwkID:   0,
					NwkAddr: 0x1020304,
				},
				FCtrl: FCtrl{
					ADR:       true,
					ADRACKReq: false,
					ACK:       false,
					FPending:  false,
					ClassB:    false,
					FOptsLen:  0,
				},
				FCnt:  179,
				FOpts: NewFOptsSet(UnconfirmedDataDown),
			},
			FPort:      7,
			FRMPayload: []byte{80, 108, 101, 97, 115, 101, 100, 32, 116, 111, 32, 109, 101, 101, 116, 32, 121, 111, 117, 32, 77, 114, 32, 83, 116, 97, 110, 108, 101, 121, 33, 33},
			// ASCII :         P   l    e    a   s    e    d        t    o        m    e    e    t        y    o    u        M   r        S   t    a   n    l    e    y    !   !
			MACCommands: NewMACCommandSet(UnconfirmedDataDown, 222),
		},
		MIC: 0,
	}
	return message
}

func TestEncryptResponseMessage(t *testing.T) {
	// First successfully server encrypted and verified/decrypted message on LoRa device
	payload := createPleasedToMeetYouMrStanleyMessage()

	appSKey, _ := AESKeyFromString("0102 0304 0506 0708 090A 0B0C 0D0E 0F10")
	nwkSKey, _ := AESKeyFromString("0102 0304 0506 0708 090A 0B0C 0D0E 0F10")

	payload.encrypt(nwkSKey, appSKey)

	// Since the MIC is calculated across the entire message, including encrypted payload, we can
	// assume that the message is ok
	if payload.MIC != 0x5345CB54 {
		logging.Debug("MIC is: %x", payload.MIC)
		t.Error("Failed encrypting message. Expected 0x5345CB54. Payload.MIC:", payload.MIC)
	}
}

func TestDecodingPartialBuffers(t *testing.T) {
	p := NewPHYPayload(ConfirmedDataUp)
	p.MACPayload.FHDR.FCnt = uint16(0xFFFF)
	p.MACPayload.FPort = 223
	p.MACPayload.FHDR.FOpts.Add(NewUplinkMACCommand(LinkADRAns))
	p.MACPayload.FHDR.FOpts.Add(NewUplinkMACCommand(BeaconFreqReq))
	p.MACPayload.FHDR.FOpts.Add(NewUplinkMACCommand(PingSlotChannelReq))
	p.MACPayload.FHDR.FOpts.Add(NewUplinkMACCommand(LinkCheckAns))
	p.MACPayload.FHDR.FOpts.Add(NewUplinkMACCommand(DevStatusReq))
	buf, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("Got error marshalling buffer: %v", err)
	}
	// This should succeed
	decoded := NewPHYPayload(Proprietary)
	if err := decoded.UnmarshalBinary(buf); err != nil {
		t.Fatalf("Got error unmarshaling a nice buffer: %v", err)
	}

	// The MIC is 4 characters long and the real contents can't be determined with a
	// payload
	for i := 0; i < len(buf)-1; i++ {
		var err error
		tmp := NewPHYPayload(Proprietary)
		if err = tmp.UnmarshalBinary(buf[0:i][:]); err == nil {
			t.Fatalf("Expected error when buffer is %d of %d bytes but got none", i, len(buf))
		}
	}
}

func TestEncodeDecodeOfFRMPayloadUnconfirmedDataDown(t *testing.T) {
	p := NewPHYPayload(UnconfirmedDataDown)
	p.MACPayload.FHDR.DevAddr = DevAddr{NwkID: 0, NwkAddr: 0x1E672E6}
	p.MACPayload.FHDR.FCtrl = FCtrl{
		ADR:       true,
		ADRACKReq: true,
		ACK:       false,
		FPending:  false,
		ClassB:    false,
		FOptsLen:  0,
	}
	p.MACPayload.FHDR.FCnt = 24
	p.MACPayload.FPort = 12
	p.MACPayload.FRMPayload = []byte{0x50, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x64, 0x20, 0x74, 0x6f, 0x20, 0x6d, 0x65, 0x65, 0x74, 0x20, 0x79, 0x6f, 0x75, 0x20, 0x4d, 0x72, 0x20, 0x53, 0x74, 0x61, 0x6e, 0x6c, 0x65, 0x79, 0x21}

	appSKey, _ := AESKeyFromString("E001 2A22 25B8 585E DCEC 7042 4798 C510")
	nwkSKey, _ := AESKeyFromString("3C5E 5C9F 469E EF3E 02CC D4FF 9531 31BA")

	buf, err := p.MarshalBinary()
	if err != nil {
		t.Error("Did not expect error when encoding message: ", err)
	}
	if buf == nil {
		t.Error("Expected buffer to be returned but it was nil")
	}

	var p2 = NewPHYPayload(Proprietary)
	err = p2.UnmarshalBinary(buf)

	if err != nil {
		t.Error("Did not expect error when unmarshalling buffer")
	}

	if !reflect.DeepEqual(p.MACPayload.FHDR, p2.MACPayload.FHDR) {
		t.Fatalf("FHDR is different: %+v != %+v", p.MACPayload.FHDR, p2.MACPayload.FHDR)
	}
	if p.MACPayload.FPort != p2.MACPayload.FPort {
		t.Fatalf("FPort is different: %+v != %+v", p.MACPayload.FPort, p2.MACPayload.FPort)
	}
	if len(p.MACPayload.FRMPayload) != len(p2.MACPayload.FRMPayload) {
		t.Logf("ENCODED: %+v   MIC: %08x", p.MACPayload.FRMPayload, p.MIC)
		t.Logf("DECODED: %+v   MIC: %08x", p2.MACPayload.FRMPayload, p.MIC)
		t.Fatal("Encoded and Decoded FRMPayload has different size")
	}

	for i := 0; i < len(p2.MACPayload.FRMPayload); i++ {
		if p.MACPayload.FRMPayload[i] != p2.MACPayload.FRMPayload[i] {
			t.Errorf("MACPayload.FRMPayload different: %+v != %+v", p, p2)
			break
		}
	}

	p2.encrypt(nwkSKey, appSKey) // Will also calculate MIC
	if p2.MIC == 0 {
		t.Fatalf("MIC not calculated correctly: %x\n", p2.MIC)
	}
	if p2.MIC != 1007493435 {
		t.Errorf("MIC calculated incorrectly: %v != %v", p2.MIC, 1007493435)
	}

	if p2.MHDR != p.MHDR {
		t.Errorf("MHDR decoded differently: %v != %v", p.MHDR, p2.MHDR)
	}

	if p2.MACPayload.FHDR.DevAddr != p.MACPayload.FHDR.DevAddr {
		t.Errorf("FHDR.DevAddr decoded differently: %v != %v", p.MACPayload.FHDR.DevAddr, p2.MACPayload.FHDR.DevAddr)
	}

	if p2.MACPayload.FHDR.FCnt != p.MACPayload.FHDR.FCnt {
		t.Errorf("FHDR.FCnt decoded differently: %v != %v", p.MACPayload.FHDR.FCnt, p2.MACPayload.FHDR.FCnt)
	}

	if p2.MACPayload.FHDR.FCtrl != p.MACPayload.FHDR.FCtrl {
		t.Errorf("FHDR.FCtrl decoded differently: %v != %v", p.MACPayload.FHDR.FCtrl, p2.MACPayload.FHDR.FCtrl)
	}

	if p2.MACPayload.FPort != p.MACPayload.FPort {
		t.Errorf("FHDR.FPort decoded differently: %v != %v", p.MACPayload.FPort, p2.MACPayload.FPort)
	}

}

func compareMACCommands(where string, a1 *MACCommandSet, a2 *MACCommandSet, t *testing.T) {
	if a1.Size() != a2.Size() {
		t.Fatalf("Lengths of MAC command arrays in %s does not match: %d != %d", where, a1.Size(), a2.Size())
	}
	if a1.Message() != a2.Message() {
		t.Fatalf("One is uplink, the other isn't")
	}

	a1list := a1.List()
	a2list := a2.List()
	for i := range a1list {
		if a1list[i].ID() != a2list[i].ID() {
			t.Fatal("Not the same commands in set")
		}
	}
}

func comparePHYPayload(p1 *PHYPayload, p2 *PHYPayload, t *testing.T) {
	if p1.MHDR != p2.MHDR {
		t.Fatalf("Marshaled and unmarshaled MHDR doesn't match: %v != %v", p1.MHDR, p2.MHDR)
	}

	if p1.MACPayload.FHDR.DevAddr != p2.MACPayload.FHDR.DevAddr {
		t.Fatalf("Marshaled and unmarshaled DevAddr doesn't match: %v != %v", p1.MACPayload.FHDR.DevAddr, p2.MACPayload.FHDR.DevAddr)
	}

	if p1.MACPayload.FHDR.FCnt != p1.MACPayload.FHDR.FCnt {
		t.Fatalf("Marshaled and unmarshaled FCnt doesn't match: %v != %v", p1.MACPayload.FHDR.FCnt, p1.MACPayload.FHDR.FCnt)
	}

	if p1.MACPayload.FHDR.FCnt != p2.MACPayload.FHDR.FCnt {
		t.Fatalf("Marshaled and unmarshaled FCnt doesn't match: %v != %v", p1.MACPayload.FHDR.FCnt, p2.MACPayload.FHDR.FCnt)
	}

	if p1.MACPayload.FHDR.FCtrl != p2.MACPayload.FHDR.FCtrl {
		t.Fatalf("Marshaled and unmarshaled FCtrl doesn't match: %v != %v", p1.MACPayload.FHDR.FCtrl, p2.MACPayload.FHDR.FCtrl)
	}

	compareMACCommands("FHDR", &p1.MACPayload.FHDR.FOpts, &p2.MACPayload.FHDR.FOpts, t)

	if p1.MACPayload.FPort != p1.MACPayload.FPort {
		t.Fatalf("FPort aren't the same: %v != %v", p1.MACPayload.FPort, p2.MACPayload.FPort)
	}

	compareMACCommands("MACPayload", &p1.MACPayload.MACCommands, &p2.MACPayload.MACCommands, t)

	if len(p1.MACPayload.FRMPayload) != len(p2.MACPayload.FRMPayload) {
		t.Fatalf("FRMPayload aren't same length: %v != %v", len(p1.MACPayload.FRMPayload), len(p2.MACPayload.FRMPayload))
	}

	for i, v := range p1.MACPayload.FRMPayload {
		if v != p2.MACPayload.FRMPayload[i] {
			t.Fatalf("FRMPayload[%d] doesnt't match: %v != %v", i, v, p2.MACPayload.FRMPayload[i])
		}
	}
}

// Do a simple encode - decode and compare the results. Strictly speaking this tests
// both encoding and decoding but it makes everything easier.
func TestEncodeDecode(t *testing.T) {
	fOpts := NewMACCommandSet(UnconfirmedDataUp, MaxFOptsLen)
	fOpts.Add(NewUplinkMACCommand(DevStatusAns))
	fOpts.Add(NewUplinkMACCommand(LinkADRAns))
	input := NewPHYPayload(UnconfirmedDataUp)
	input.MACPayload.FHDR.DevAddr = DevAddr{NwkID: 1, NwkAddr: 2}
	input.MACPayload.FHDR.FCtrl = FCtrl{
		ADR:       true,
		ACK:       true,
		ADRACKReq: false,
		FPending:  true,
		ClassB:    true,
		FOptsLen:  0,
	}
	input.MACPayload.FHDR.FCnt = 12
	input.MACPayload.FHDR.FOpts = fOpts
	input.MACPayload.FPort = 12
	input.MACPayload.FRMPayload = []byte{1, 2, 3, 4, 5, 6}

	buffer, err := input.MarshalBinary()
	if err != nil {
		t.Fatal("Got error marshalling binary: ", err)
	}

	output := NewPHYPayload(Proprietary)
	if err = output.UnmarshalBinary(buffer); err != nil {
		t.Fatal("Got error unmarshaling binary: ", err)
	}

	comparePHYPayload(&input, &output, t)
}

// Encode something with MAC commands in the payload (and no commands piggybacked)
func TestEncodeMACPayload(t *testing.T) {
	macCommands := NewMACCommandSet(UnconfirmedDataUp, 222)
	macCommands.Add(NewUplinkMACCommand(DevStatusAns))
	macCommands.Add(NewUplinkMACCommand(LinkADRAns))
	input := NewPHYPayload(UnconfirmedDataUp)
	input.MACPayload.FHDR.DevAddr = DevAddr{NwkID: 1, NwkAddr: 2}
	input.MACPayload.FHDR.FCtrl = FCtrl{
		ADR:       true,
		ACK:       true,
		ADRACKReq: false,
		FPending:  true,
		ClassB:    true,
		FOptsLen:  0,
	}
	input.MACPayload.FHDR.FCnt = 12
	input.MACPayload.FPort = 12
	input.MACPayload.MACCommands = macCommands

	buffer, err := input.MarshalBinary()
	if err != nil {
		t.Fatal("Error marshaling binary: ", err)
	}
	if input.MACPayload.FPort != 0 {
		t.Fatalf("Expected port to be 0 but it is %d", input.MACPayload.FPort)
	}

	output := NewPHYPayload(Proprietary)
	if err = output.UnmarshalBinary(buffer); err != nil {
		t.Fatal("Error unmarshaling binary: ", err)
	}

	comparePHYPayload(&input, &output, t)
}

// Encode something with no payload, just MAC commands in the header
func TestEncodeMACPiggybackNoPayload(t *testing.T) {
	input := NewPHYPayload(UnconfirmedDataUp)

	input.MACPayload.FHDR.DevAddr = DevAddr{NwkID: 1, NwkAddr: 2}
	input.MACPayload.FHDR.FCtrl = FCtrl{
		ADR:       true,
		ACK:       true,
		ADRACKReq: false,
		FPending:  true,
		ClassB:    true,
		FOptsLen:  0,
	}
	input.MACPayload.FHDR.FCnt = 12
	input.MACPayload.FHDR.FOpts.Add(NewUplinkMACCommand(DevStatusAns))
	input.MACPayload.FHDR.FOpts.Add(NewUplinkMACCommand(LinkADRAns))

	input.MACPayload.FPort = 12

	buffer, err := input.MarshalBinary()
	if err != nil {
		t.Fatal("Error marshaling binary: ", err)
	}

	// Port should be set to 0 (and omitted since there's no payload)
	if input.MACPayload.FPort != 0 {
		t.Fatalf("Expected port to be 0 but it is %d", input.MACPayload.FPort)
	}

	output := NewPHYPayload(Proprietary)

	if err = output.UnmarshalBinary(buffer); err != nil {
		t.Fatal("Error unmarshaling binary: ", err)
	}

	comparePHYPayload(&input, &output, t)
}

func TestInvalidPortEncoding(t *testing.T) {
	input := NewPHYPayload(Proprietary)
	input.MACPayload.FPort = 0
	input.MACPayload.FRMPayload = []byte{1, 2, 3, 4, 5}

	if _, err := input.MarshalBinary(); err == nil {
		t.Fatal("Expected error when using port = 0 with payload")
	}

	input.MACPayload.FPort = 224
	if _, err := input.MarshalBinary(); err == nil {
		t.Fatal("Expected error when port > 223")
	}
}

func TestEncodeJoinAccept(t *testing.T) {
	// Encode JoinAccept message then test the output
	appKey, _ := AESKeyFromString("00010203 04050607 00010203 04050607")

	p := NewPHYPayload(JoinAccept)
	p.JoinAcceptPayload = JoinAcceptPayload{
		AppNonce:   [3]byte{0, 1, 2},
		NetID:      0x01020304,
		DevAddr:    DevAddr{NwkID: 1, NwkAddr: 2},
		DLSettings: DLSettings{RX1DRoffset: 3, RX2DataRate: 4},
		RxDelay:    5,
		CFList:     CFList{},
	}

	buffer, err := p.EncodeJoinAccept(appKey)
	if err != nil {
		t.Fatal("Got error marshalling binary: ", err)
	}
	if buffer == nil {
		t.Fatal("Did not get a buffer")
	}

	p2 := NewPHYPayload(ConfirmedDataUp)
	if err := p2.UnmarshalBinary(buffer); err != nil {
		t.Fatal("Got error unmarshaling binary: ", err)
	}
}

// Proprietary and JoinAccept/JoinRequest messages can't be marshaled by
// MarshalBinary
func TestUnmarshableMessageTypes(t *testing.T) {
	p := NewPHYPayload(JoinAccept)

	if _, err := p.MarshalBinary(); err == nil {
		t.Fatal("Expected JoinAccept message to raise error")
	}

	p.MHDR.MType = JoinRequest
	if _, err := p.MarshalBinary(); err == nil {
		t.Fatal("Expected JoinRequest message to raise error")
	}

	p.MHDR.MType = Proprietary
	if _, err := p.MarshalBinary(); err == nil {
		t.Fatal("Expected Proprietary message to raise error")
	}
}

func TestProprietaryUnmarshal(t *testing.T) {
	// Make a proprietary message buffer
	mhdr := MHDR{MType: Proprietary, MajorVersion: LoRaWANR1}
	pos := 0
	buf := make([]byte, 1024)
	mhdr.encode(buf, &pos)

	p := NewPHYPayload(ConfirmedDataUp)
	if err := p.UnmarshalBinary(buf); err != ErrInvalidMessageType {
		t.Fatalf("Expected error when unmarshaling proprietary message: %v", err)
	}

	mhdr.MType = ConfirmedDataDown
	mhdr.MajorVersion = 0xFF
	pos = 0
	if err := mhdr.encode(buf, &pos); err != nil {
		t.Fatalf("Couldn't encode buffer: %v", err)
	}
	if err := p.UnmarshalBinary(buf); err != ErrInvalidLoRaWANVersion {
		t.Fatalf("Expected error when unmarshaling proprietary message: %v", err)
	}
}

func TestUnmarshalPartialBuffers(t *testing.T) {
	p := NewPHYPayload(ConfirmedDataDown)

	buf, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("Couldn't marshal: %v", err)
	}
	for i := 0; i < len(buf)-1; i++ {
		if err := p.UnmarshalBinary(buf[0:i]); err != ErrBufferTruncated {
			t.Fatalf("Expected error when unmarshaling short buffer. Got %v", err)
		}
	}
}

func TestDecodeJoinAcceptInvalidMIC(t *testing.T) {
	p := NewPHYPayload(JoinAccept)
	p.JoinAcceptPayload = JoinAcceptPayload{
		AppNonce: [3]byte{0, 1, 2},
		NetID:    0x00010203,
		DevAddr:  DevAddr{NwkID: 1, NwkAddr: 2},
		DLSettings: DLSettings{
			RX1DRoffset: 1,
			RX2DataRate: 2,
		},
		RxDelay: 4,
		CFList:  CFList{},
	}
	appKey, _ := NewAESKey()
	buf, err := p.EncodeJoinAccept(appKey)
	if err != nil {
		t.Fatalf("Error encoding JoinAccept: %v", err)
	}

	// Modify the MIC bytes
	buf[len(buf)-4] = 0
	buf[len(buf)-3] = 0
	buf[len(buf)-2] = 0
	buf[len(buf)-1] = 0

	if err := p.DecodeJoinAccept(appKey, buf); err != ErrInvalidMIC {
		t.Fatalf("Expected invalid mic error message but got %v", err)
	}
}

func TestDecodeJoinAcceptInvalidMessageType(t *testing.T) {
	p := NewPHYPayload(JoinRequest)
	key, _ := NewAESKey()
	if err := p.DecodeJoinAccept(key, make([]byte, 1)); err == nil {
		t.Fatal("Expected error when encoding joinaccept with invalid message type")
	}
}
