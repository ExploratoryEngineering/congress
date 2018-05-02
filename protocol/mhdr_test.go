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
import "testing"

func TestMHDRMessageTypes(t *testing.T) {
	mhdr := MHDR{}
	buf := make([]byte, 1)
	messageTypes := []MType{
		JoinRequest,
		JoinAccept,
		UnconfirmedDataDown,
		UnconfirmedDataUp,
		ConfirmedDataUp,
		ConfirmedDataDown,
		RFU,
		Proprietary}
	for _, mtype := range messageTypes {
		pos := 0
		mhdr.MType = mtype
		mhdr.MajorVersion = MaxSupportedVersion
		if err := mhdr.encode(buf, &pos); err != nil {
			t.Error("Got error encoding MType ", mtype, ": ", err)
		}
		pos = 0
		if err := mhdr.decode(buf, &pos); (mtype != RFU && mtype != Proprietary) && err != nil {
			t.Error("Got error decoding MType ", mtype, ": ", err)
		}
		if mhdr.MType != mtype {
			t.Error("Decoded message doesn't match encoded for MType ", mtype)
		}
	}
}

func TestMHDRInvalidVersion(t *testing.T) {
	buffer := make([]byte, 1)
	buffer[0] = 0x0F // This will match the mtype field (1) but the version will be 11 (aka 3)
	mhdr := MHDR{}
	pos := 0
	if err := mhdr.decode(buffer, &pos); err == nil {
		t.Error("Expected error when decoding invalid MHDR version but didn't get one.")
	}
}

func TestEncodeDecodeMHDR(t *testing.T) {
	m1 := MHDR{MType: 1, MajorVersion: 0}
	m2 := MHDR{MType: 2, MajorVersion: MaxSupportedVersion}
	m3 := MHDR{MType: 3, MajorVersion: 0}

	test := func(in MHDR) {
		buffer := make([]byte, 10)
		pos := 0
		if err := in.encode(buffer, &pos); err != nil {
			t.Fatal("Error encoding MHDR: ", err)
		}
		d := MHDR{}
		pos = 0
		if err := d.decode(buffer, &pos); err != nil {
			t.Fatal("Error decoding MHDR: ", err)
		}
		if in != d {
			t.Fatalf("Did not encode/decode to same: %v != %v", in, d)
		}
	}
	test(m1)
	test(m2)
	test(m3)
}

func TestMTypeStringer(t *testing.T) {
	for _, v := range []MType{JoinAccept, JoinRequest, ConfirmedDataDown, ConfirmedDataUp, UnconfirmedDataDown, UnconfirmedDataUp, Proprietary, RFU} {
		t.Log(v.String())
	}
}
