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

// Tests for the class A MAC commands
func macCommandStandardTests(cmd MACCommand, kind CID, t *testing.T) {

	if cmd.ID() != kind {
		t.Errorf("%v reports incorrect type", cmd)
	}

	var emptyBuffer []byte
	pos := 0
	err := cmd.encode(nil, &pos)
	if err == nil {
		t.Errorf("%v doesn't check the buffer", cmd)
	}

	err = cmd.encode(emptyBuffer, &pos)
	if err == nil {
		t.Errorf("%v doesn't check the buffer length", cmd)
	}

	buffer := make([]byte, 10)
	invalidPos := 11
	err = cmd.encode(buffer, &invalidPos)
	if err == nil {
		t.Errorf("%v doesn't check the upper bound index", cmd)
	}
	basicDecoderTests(t, cmd)
	basicEncoderTests(t, cmd)
}

func TestLinkCheckReq(t *testing.T) {
	m := MACLinkCheckReq{macBase{LinkCheckReq, true}}
	macCommandStandardTests(&m, LinkCheckReq, t)

	buffer := make([]byte, 1)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Couldn't encode LinkCheckReq")
	}

	p := MACLinkCheckReq{macBase{LinkCheckReq, true}}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Couldn't decode LinkCheckReq")
	}

	if dpos != pos {
		t.Errorf("LinkCheckReq decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestLinkCheckAns(t *testing.T) {
	m := MACLinkCheckAns{macBase{LinkCheckAns, true}, 12, 16}
	macCommandStandardTests(&m, LinkCheckAns, t)

	buffer := make([]byte, 3)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Couldn't encode LinkCheckAns")
	}

	p := MACLinkCheckAns{macBase{LinkCheckAns, true}, 0, 0}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Couldn't decode LinkCheckAns")
	}
	if p.Margin != m.Margin || p.GwCnt != m.GwCnt {
		t.Errorf("LinkCheckAns values do not match %v != %v", p, m)
	}

	if dpos != pos {
		t.Errorf("LinkCheckAns decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestLinkADRReq(t *testing.T) {
	m := MACLinkADRReq{macBase{LinkADRReq, true}, 0x01, 0x02, 0x0304, 0x05}
	macCommandStandardTests(&m, LinkADRReq, t)

	buffer := make([]byte, 5)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode LinkADRReq")
	}

	p := MACLinkADRReq{macBase{LinkADRReq, true}, 0, 0, 0, 0}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode LinkADRReq")
	}
	if p.DataRate != m.DataRate || p.TXPower != m.TXPower || p.ChMask != m.ChMask || p.Redundancy != m.Redundancy {
		t.Errorf("LinkADRReq values do not match %v != %v", p, m)
	}
	if dpos != pos {
		t.Errorf("LinkADRReq decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestLinkADRAns(t *testing.T) {
	m := MACLinkADRAns{macBase{LinkADRAns, true}, true, true, true}
	macCommandStandardTests(&m, LinkADRAns, t)

	buffer := make([]byte, 2)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode LinkADRAns")
	}

	p := MACLinkADRAns{macBase{LinkADRAns, true}, false, false, false}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode LinkADRAns")
	}
	if p != m {
		t.Errorf("LinkADRAns values do not match %v != %v", p, m)
	}

	if dpos != pos {
		t.Errorf("LinkADRAns decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestDutyCycleReq(t *testing.T) {
	m := MACDutyCycleReq{macBase{DutyCycleReq, false}, 0xED}
	macCommandStandardTests(&m, DutyCycleReq, t)

	buffer := make([]byte, 2)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode DutyCycleReq")
	}

	p := MACDutyCycleReq{macBase{DutyCycleReq, false}, 0}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode DutyCycleReq")
	}

	if p.MaxDCycle != m.MaxDCycle {
		t.Errorf("DutyCycleReq values do not match %v != %v", p, m)
	}

	if dpos != pos {
		t.Errorf("DutyCycleReq decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestDutyCycleAns(t *testing.T) {
	m := MACDutyCycleAns{macBase{DutyCycleAns, false}}
	macCommandStandardTests(&m, DutyCycleAns, t)

	buffer := make([]byte, 1)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode DutyCycleAns")
	}

	p := MACDutyCycleAns{macBase{DutyCycleAns, false}}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode DutyCycleAns")
	}

	if dpos != pos {
		t.Errorf("DutyCycleReq decodes different number of bytes (%d != %d)", dpos, pos)
	}

}

func TestRXParamSetupReq(t *testing.T) {
	m := MACRXParamSetupReq{macBase{RXParamSetupReq, true}, 0x1, 0x2, 0xCDEF01}
	macCommandStandardTests(&m, RXParamSetupReq, t)

	buffer := make([]byte, 5)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode RXParamSetupReq")
	}

	p := MACRXParamSetupReq{macBase{RXParamSetupReq, true}, 0, 0, 0}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode RXParamSetupReq")
	}

	if p != m {
		t.Errorf("RXParamSetupReq values do not match: %v != %v", p, m)
	}

	if dpos != pos {
		t.Errorf("RXParamSetupReq decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestRXParamSetupAns(t *testing.T) {
	m := MACRXParamSetupAns{macBase{RXParamSetupAns, false}, true, false, true}
	macCommandStandardTests(&m, RXParamSetupAns, t)

	buffer := make([]byte, 2)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode RXParamSetupAns")
	}

	p := MACRXParamSetupAns{macBase{RXParamSetupAns, false}, false, false, false}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode RXParamSetupAns")
	}

	if p != m {
		t.Errorf("RXParamSetupAns values do not match: %v != %v", p, m)
	}

	if dpos != pos {
		t.Errorf("RXParamSetupAns decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestDevStatusReq(t *testing.T) {
	m := MACDevStatusReq{macBase{DevStatusReq, false}}
	macCommandStandardTests(&m, DevStatusReq, t)

	buffer := make([]byte, 1)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode DevStatusReq: ", err)
	}

	p := MACDevStatusReq{macBase{DevStatusReq, false}}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode DevStatusReq: ", err)
	}

	if dpos != pos {
		t.Errorf("DevStatusReq decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestDevStatusAns(t *testing.T) {
	m := MACDevStatusAns{macBase{DevStatusAns, true}, 0xA, 0xB}
	macCommandStandardTests(&m, DevStatusAns, t)

	buffer := make([]byte, 3)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode DevStatusAns: ", err)
	}

	p := MACDevStatusAns{macBase{DevStatusAns, true}, 0, 0}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode DevStatusAns")
	}
	if p != m {
		t.Errorf("Encoded and decoded DevStatus ans are different: %v != %v", p, m)
	}

	if dpos != pos {
		t.Errorf("DevStatusAns decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestNewChannelReq(t *testing.T) {
	m := MACNewChannelReq{macBase{NewChannelReq, false}, 0x01, 0x020304, 0x05, 0x06}
	macCommandStandardTests(&m, NewChannelReq, t)

	buffer := make([]byte, 6)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode NewChannelReq: ", err)
	}

	p := MACNewChannelReq{macBase{NewChannelReq, false}, 0, 0, 0, 0}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode NewChannelReq: ", err)
	}

	if m != p {
		t.Errorf("Encoded and decoded NewChannelReq are different: %v != %v", p, m)
	}

	if dpos != pos {
		t.Errorf("NewChannelReq decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestNewChannelAns(t *testing.T) {
	m := MACNewChannelAns{macBase{NewChannelAns, false}, true, false}
	macCommandStandardTests(&m, NewChannelAns, t)

	buffer := make([]byte, 2)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode NewChannelAns: ", err)
	}

	p := MACNewChannelAns{macBase{NewChannelAns, false}, false, false}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode NewChannelAns: ", err)
	}

	if p != m {
		t.Errorf("Encoded and decoded NewChannelAns are different: %v != %v", p, m)
	}

	if dpos != pos {
		t.Errorf("NewChannelAns decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestRXTimingSetupReq(t *testing.T) {
	m := MACRXTimingSetupReq{macBase{RXTimingSetupReq, false}, 8}
	macCommandStandardTests(&m, RXTimingSetupReq, t)

	buffer := make([]byte, 2)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode RXTimingSetupReq: ", err)
	}

	p := MACRXTimingSetupReq{macBase{RXTimingSetupReq, false}, 0}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode RXTimingSetupReq: ", err)
	}

	if p != m {
		t.Errorf("Encoded and decoded RXTimingSetupReq are different: %v != %v", p, m)
	}

	if dpos != pos {
		t.Errorf("RXTimingSetupReq decodes different number of bytes (%d != %d)", dpos, pos)
	}
}

func TestRXTimingSetupAns(t *testing.T) {
	m := MACRXTimingSetupAns{macBase{RXTimingSetupAns, true}}
	macCommandStandardTests(&m, RXTimingSetupAns, t)

	buffer := make([]byte, 1)
	pos := 0
	err := m.encode(buffer, &pos)
	if err != nil {
		t.Error("Could not encode RXTImingSetupAns: ", err)
	}

	p := MACRXTimingSetupAns{macBase{RXTimingSetupAns, true}}
	dpos := 0
	err = p.decode(buffer, &dpos)
	if err != nil {
		t.Error("Could not decode RXTimingSetupAns: ", err)
	}

	if dpos != pos {
		t.Errorf("RXTimingSetupAns decodes different number of bytes (%d != %d)", dpos, pos)
	}
}
