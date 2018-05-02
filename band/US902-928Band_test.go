package band

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

func TestNameUS(t *testing.T) {
	b := newUS902()
	if b.Name() != "US 902-928MHz ISM Band" {
		t.Error("Unexpected band name")
	}
}

func TestAckTimeoutUS(t *testing.T) {
	b := newUS902()

	for i := 0; i < 1000; i++ {
		timeout := b.Configuration().AckTimeout()
		if (timeout < 1) || (timeout > 3) {
			t.Errorf("Receivd an unexpected value from AckTimeout")
		}
	}

}

func TestDefaultConfigurationUS(t *testing.T) {
	b := newUS902()

	if b.Configuration().ReceiveDelay1 != 1 {
		t.Errorf("Wrong default RECEIVE_DELAY1 for %s [7.2.8]", b.Name())
	}
	if b.Configuration().ReceiveDelay2 != b.Configuration().ReceiveDelay1+1 {
		t.Errorf("Wrong default RECEIVE_DELAY2 for %s [7.2.8]", b.Name())
	}
	if b.Configuration().JoinAccepDelay1 != 5 {
		t.Errorf("Wrong default JOIN_ACCEPT_DELAY1 for %s [7.2.8]", b.Name())
	}
	if b.Configuration().JoinAccepDelay2 != 6 {
		t.Errorf("Wrong default JOIN_ACCEPT_DELAY2 for %s [7.2.8]", b.Name())
	}
	if b.Configuration().MaxFCntGap != 16384 {
		t.Errorf("Wrong default MAX_FCNT_GAP for %s [7.2.8]", b.Name())
	}
	if b.Configuration().AdrAckLimit != 64 {
		t.Errorf("Wrong default ADR_ACK_LIMIT for %s [7.2.8]", b.Name())
	}
	if b.Configuration().AdrAckDelay != 32 {
		t.Errorf("Wrong default ADR_ACK_DELAY for %s [7.2.8]", b.Name())
	}
	if b.Configuration().DefaultTxPower != 20 {
		t.Errorf("Wrong default TXPower for %s [7.2.2]", b.Name())
	}
	if b.Configuration().SupportsJoinAcceptCFList {
		t.Errorf("Wrong default value for SupportsJoinAcceptCFList for %s [7.2.4]", b.Name())
	}
	if b.Configuration().RX2Frequency != 923.3 {
		t.Errorf("Wrong default RX2Frequency for %s [7.2.7]", b.Name())
	}
	if b.Configuration().RX2DataRate != 8 {
		t.Errorf("Wrong default RX2DataRate for %s [7.2.7]", b.Name())
	}
}

func TestTxPowerUS(t *testing.T) {
	b := newUS902()

	testParams := []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	expectedOutput := []int8{30, 28, 26, 24, 22, 20, 18, 16, 14, 12, 10}

	for i := 0; i < len(testParams); i++ {
		power, err := b.TxPower(testParams[i])
		if err != nil {
			t.Errorf("%s, %s [7.2.3]", err, b.Name())
		}
		if power != expectedOutput[i] {
			t.Errorf("Wrong TxPower configuration for %d, %s [7.2.3]", power, b.Name())
		}
	}

	_, err := b.TxPower(42)
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.2.3]", b.Name())
	}

}

func TestEncodingUS(t *testing.T) {
	b := newUS902()
	testParams := []uint8{0, 1, 2, 3, 4, 8, 9, 10, 11, 12, 13}
	expectedSpreadFactors := []uint8{10, 9, 8, 7, 8, 12, 11, 10, 9, 8, 7}
	expectedBandwidths := []uint32{125, 125, 125, 125, 500, 500, 500, 500, 500, 500, 500}
	expectedBitrates := []uint32{980, 1760, 3125, 5470, 12500, 980, 1760, 3900, 7000, 12500, 21900}

	for i := 0; i < len(testParams); i++ {
		encoding, err := b.Encoding(testParams[i])
		if err != nil {
			t.Errorf("%s, %s [7.1.3]", err, b.Name())
		}
		if encoding.Modulation != LoRa {
			t.Errorf("Unexpected modulation (%v) for datarate (%d) %s [7.2.3]", encoding.Modulation, i, b.Name())
		}
		if encoding.SpreadFactor != expectedSpreadFactors[i] {
			t.Errorf("Unexpected spreadfactor (%v) for datarate (%d) %s [7.2.3]", encoding.SpreadFactor, i, b.Name())
		}
		if encoding.Bandwidth != expectedBandwidths[i] {
			t.Errorf("Unexpected bandwidth (%v) for datarate (%d) %s [7.2.3]", encoding.Bandwidth, i, b.Name())
		}
		if encoding.BitRate != expectedBitrates[i] {
			t.Errorf("Unexpected bit rate (%v) for datarate (%d) %s [7.2.3]", encoding.BitRate, i, b.Name())
		}
	}

	var dr uint8
	for dr = 5; dr < 7; dr++ {
		_, err := b.Encoding(dr)
		if err == nil {
			t.Errorf("Invalid parameter should fail, %s [7.2.3]", b.Name())
		}
	}

	_, err := b.Encoding(42)
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.2.3]", b.Name())
	}
}

func TestMaximumPayloadUS(t *testing.T) {
	b := newUS902()
	testParams := []string{"SF10BW125", "SF9BW125", "SF8BW125", "SF7BW125", "SF8BW500", "SF12BW500", "SF11BW500", "SF10BW500", "SF9BW500", "SF7BW500"}
	expectedMs := []uint8{19, 61, 137, 250, 250, 41, 117, 230, 230, 230, 230}
	expectedNs := []uint8{11, 53, 129, 242, 242, 33, 109, 222, 222, 222, 222}

	for i := 0; i < len(testParams); i++ {
		mp, err := b.MaximumPayload(testParams[i])
		if err != nil {
			t.Errorf("%s, %s [7.2.6]", err, b.Name())
		}
		if mp.M != expectedMs[i] {
			t.Errorf("Unexpected M (%d) for datarate (%s) %s [7.2.6]", mp.M, testParams[i], b.Name())
		}
		if mp.WithoutFOpts() != expectedMs[i] {
			t.Errorf("Unexpected M (%d) for datarate (%s) %s [7.2.6]", mp.M, testParams[i], b.Name())
		}
		if mp.N != expectedNs[i] {
			t.Errorf("Unexpected N (%d) for datarate (%s) %s [7.2.6]", mp.N, testParams[i], b.Name())
		}
		if mp.WithFOpts() != expectedNs[i] {
			t.Errorf("Unexpected N (%d) for datarate (%s) %s [7.2.6]", mp.N, testParams[i], b.Name())
		}
	}

	_, err := b.MaximumPayload("SF1BW1000")
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.2.6]", b.Name())
	}
}

func TestDownlinkDataRatesUS(t *testing.T) {
	b := newUS902()
	expectedRates := [][]uint8{
		{10, 9, 8, 8},    // DR0
		{11, 10, 9, 8},   // DR1
		{12, 11, 10, 9},  // DR2
		{13, 12, 11, 10}, // DR3
		{13, 13, 12, 11}, // DR4
		{0, 0, 0, 0},     // DR5 - Invalid. Not used
		{0, 0, 0, 0},     // DR6 - Invalid. Not used
		{0, 0, 0, 0},     // DR7 - Invalid. Not used
		{8, 8, 8, 8},     // DR8
		{9, 8, 8, 8},     // DR9
		{10, 9, 8, 8},    // DR10
		{11, 10, 9, 8},   // DR11
		{12, 11, 10, 9},  // DR12
		{13, 12, 11, 10}, // DR13
	}
	datarates := []uint8{0, 1, 2, 3, 4, 8, 9, 10, 11, 12, 13}
	var UpstreamDataRate uint8
	var RX1DROffset uint8
	for rateIndex := 0; rateIndex < len(datarates); rateIndex++ {
		for RX1DROffset = 0; RX1DROffset <= 3; RX1DROffset++ {
			rate, err := b.downlinkDataRate(datarates[rateIndex], RX1DROffset)
			if err != nil {
				t.Errorf("%s, %s [7.2.7]", err, b.Name())
			}
			if rate != expectedRates[datarates[rateIndex]][RX1DROffset] {
				t.Errorf("Unexpected downstream datarate (%d) for given upstream datarate/RX1DROffset (%d/%d) %s [7.2.7]", rate, UpstreamDataRate, RX1DROffset, b.Name())
			}
		}
	}

	var dr uint8
	for dr = 5; dr < 7; dr++ {
		_, err := b.downlinkDataRate(dr, 0)
		if err == nil {
			t.Errorf("Invalid parameter should fail, %s [7.2.7]", b.Name())
		}
	}

	_, err := b.downlinkDataRate(99, 0)
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.2.7]", b.Name())
	}
	_, err = b.downlinkDataRate(99, 0)
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.2.7]", b.Name())
	}
	_, err = b.downlinkDataRate(0, 99)
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.2.7]", b.Name())
	}
}

func TestGetRX1ParametersUS(t *testing.T) {
	b := newUS902()
	dlParams, err := b.GetRX1Parameters(63, 0 /* only channel is used in US lookup */, 3, 3)
	if err != nil {
		t.Error(err)
	}
	if dlParams.DataRate != 10 {
		t.Errorf("Unexpected data rate : %d", dlParams.DataRate)
	}
	if dlParams.Frequency != 927.5 {
		t.Errorf("Unexpected frequency: %f", dlParams.Frequency)
	}

	dlParams, err = b.GetRX1Parameters(0, 868.1, 30, 3)
	if err == nil {
		t.Errorf("Expected invalid data rate.")
	}
	dlParams, err = b.GetRX1Parameters(0, 868.1, 3, 30)
	if err == nil {
		t.Errorf("Expected invalid data rate offset.")
	}
}

func TestGetDataRateUS(t *testing.T) {
	b := newUS902()
	var dr uint8
	var err error

	dr, err = b.GetDataRate("SF10BW125")
	if (dr != 0) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF9BW125")
	if (dr != 1) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF8BW125")
	if (dr != 2) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF7BW125")
	if (dr != 3) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF8BW500")
	if (dr != 4) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF12BW500")
	if (dr != 8) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF11BW500")
	if (dr != 9) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF10BW500")
	if (dr != 10) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF9BW500")
	if (dr != 11) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF7BW500")
	if (dr != 13) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}

	dr, err = b.GetDataRate("XYZZY")
	if err == nil {
		t.Error("Expected lookup of XYZZY to fail")
	}
}
