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

func TestNameEU(t *testing.T) {
	b := newEU868()
	if b.Name() != "EU 863-870MHz ISM Band" {
		t.Error("Unexpected band name")
	}
}

func TestAckTimeoutEU(t *testing.T) {
	b := newEU868()

	for i := 0; i < 1000; i++ {
		timeout := b.Configuration().AckTimeout()
		if (timeout < 1) || (timeout > 3) {
			t.Errorf("Receivd an unexpected value from AckTimeout")
		}
	}

}

func TestDefaultConfigurationEU(t *testing.T) {
	b := newEU868()

	if b.Configuration().ReceiveDelay1 != 1 {
		t.Errorf("Wrong default RECEIVE_DELAY1 for %s [7.1.8]", b.Name())
	}
	if b.Configuration().ReceiveDelay2 != b.Configuration().ReceiveDelay1+1 {
		t.Errorf("Wrong default RECEIVE_DELAY2 for %s [7.1.8]", b.Name())
	}
	if b.Configuration().JoinAccepDelay1 != 5 {
		t.Errorf("Wrong default JOIN_ACCEPT_DELAY1 for %s [7.1.8]", b.Name())
	}
	if b.Configuration().JoinAccepDelay2 != 6 {
		t.Errorf("Wrong default JOIN_ACCEPT_DELAY2 for %s [7.1.8]", b.Name())
	}
	if b.Configuration().MaxFCntGap != 16384 {
		t.Errorf("Wrong default MAX_FCNT_GAP for %s [7.1.8]", b.Name())
	}
	if b.Configuration().AdrAckLimit != 64 {
		t.Errorf("Wrong default ADR_ACK_LIMIT for %s [7.1.8]", b.Name())
	}
	if b.Configuration().AdrAckDelay != 32 {
		t.Errorf("Wrong default ADR_ACK_DELAY for %s [7.1.8]", b.Name())
	}
	if b.Configuration().DefaultTxPower != 14 {
		t.Errorf("Wrong default TXPower for %s [7.1.2]", b.Name())
	}
	if !b.Configuration().SupportsJoinAcceptCFList {
		t.Errorf("Wrong default value for SupportsJoinAcceptCFList for %s [7.1.4]", b.Name())
	}
	if b.Configuration().RX2Frequency != 869.525 {
		t.Errorf("Wrong default RX2Frequency for %s [7.1.7]", b.Name())
	}
	if b.Configuration().RX2DataRate != 0 {
		t.Errorf("Wrong default RX2DataRate for %s [7.1.7]", b.Name())
	}
}

func TestTxPowerEU(t *testing.T) {
	b := newEU868()

	testParams := []uint8{0, 1, 2, 3, 4, 5}
	expectedOutput := []int8{20, 14, 11, 8, 5, 2}

	for i := 0; i < len(testParams); i++ {
		power, err := b.TxPower(testParams[i])
		if err != nil {
			t.Errorf("%s, %s [7.1.3]", err, b.Name())
		}
		if power != expectedOutput[i] {
			t.Errorf("Wrong TxPower configuration for %d, %s [7.1.3]", power, b.Name())
		}
	}

	_, err := b.TxPower(42)
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.1.3]", b.Name())
	}

}

func TestEncodingEU(t *testing.T) {
	b := newEU868()
	testParams := []uint8{0, 1, 2, 3, 4, 5, 6, 7}
	expectedModulations := []ModulationType{LoRa, LoRa, LoRa, LoRa, LoRa, LoRa, LoRa, FSK}
	expectedSpreadFactors := []uint8{12, 11, 10, 9, 8, 7, 7, 0}
	expectedBandwidths := []uint32{125, 125, 125, 125, 125, 125, 250, 0}
	expectedBitrates := []uint32{250, 440, 980, 1760, 3125, 5470, 11000, 50000}

	for i := 0; i < len(testParams); i++ {
		encoding, err := b.Encoding(testParams[i])
		if err != nil {
			t.Errorf("%s, %s [7.1.3]", err, b.Name())
		}
		if encoding.Modulation != expectedModulations[i] {
			t.Errorf("Unexpected modulation (%v) for datarate (%d) %s [7.1.3]", encoding.Modulation, i, b.Name())
		}
		if encoding.SpreadFactor != expectedSpreadFactors[i] {
			t.Errorf("Unexpected spreadfactor (%v) for datarate (%d) %s [7.1.3]", encoding.SpreadFactor, i, b.Name())
		}
		if encoding.Bandwidth != expectedBandwidths[i] {
			t.Errorf("Unexpected bandwidth (%v) for datarate (%d) %s [7.1.3]", encoding.Bandwidth, i, b.Name())
		}
		if encoding.BitRate != expectedBitrates[i] {
			t.Errorf("Unexpected bit rate (%v) for datarate (%d) %s [7.1.3]", encoding.BitRate, i, b.Name())
		}
	}

	_, err := b.Encoding(42)
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.1.3]", b.Name())
	}
}

func TestMaximumPayloadEU(t *testing.T) {
	b := newEU868()
	testParams := []string{"SF12BW125", "SF11BW125", "SF10BW125", "SF9BW125", "SF8BW125", "SF7BW125", "SF7BW250"}
	expectedMs := []uint8{59, 59, 59, 123, 230, 230, 230, 230}
	expectedNs := []uint8{51, 51, 51, 115, 222, 222, 222, 222}

	for i := 0; i < len(testParams); i++ {
		mp, err := b.MaximumPayload(testParams[i])
		if err != nil {
			t.Errorf("%s, %s [7.1.6]", err, b.Name())
		}
		if mp.M != expectedMs[i] {
			t.Errorf("Unexpected M (%d) for datarate (%d) %s [7.1.6]", mp.M, i, b.Name())
		}
		if mp.WithoutFOpts() != expectedMs[i] {
			t.Errorf("Unexpected M (%d) for datarate (%d) %s [7.1.6]", mp.M, i, b.Name())
		}
		if mp.N != expectedNs[i] {
			t.Errorf("Unexpected M (%d) for datarate (%d) %s [7.1.6]", mp.N, i, b.Name())
		}
		if mp.WithFOpts() != expectedNs[i] {
			t.Errorf("Unexpected M (%d) for datarate (%d) %s [7.1.6]", mp.N, i, b.Name())
		}
	}

	_, err := b.MaximumPayload("SF19BW1")
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.1.3]", b.Name())
	}
}

func TestDownlinkDataRatesEU(t *testing.T) {
	expectedRates := [][]uint8{
		{0, 0, 0, 0, 0, 0},
		{1, 0, 0, 0, 0, 0},
		{2, 1, 0, 0, 0, 0},
		{3, 2, 1, 0, 0, 0},
		{4, 3, 2, 1, 0, 0},
		{5, 4, 3, 2, 1, 0},
		{6, 5, 4, 3, 2, 1},
		{7, 6, 5, 4, 3, 2},
	}
	b := newEU868()
	var UpstreamDataRate uint8
	var RX1DROffset uint8
	for UpstreamDataRate = 0; UpstreamDataRate <= 7; UpstreamDataRate++ {
		for RX1DROffset = 0; RX1DROffset <= 5; RX1DROffset++ {
			rate, err := b.downlinkDataRate(UpstreamDataRate, RX1DROffset)
			if err != nil {
				t.Errorf("%s, %s [7.1.7]", err, b.Name())
			}
			if rate != expectedRates[UpstreamDataRate][RX1DROffset] {
				t.Errorf("Unexpected downstream datarate (%d) for given upstream datarate/RX1DROffset (%d/%d) %s [7.1.7]", rate, UpstreamDataRate, RX1DROffset, b.Name())
			}
		}
	}

	_, err := b.downlinkDataRate(99, 0)
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.1.7]", b.Name())
	}
	_, err = b.downlinkDataRate(0, 99)
	if err == nil {
		t.Errorf("Invalid parameter should fail, %s [7.1.7]", b.Name())
	}
}

func TestGetRX1ParametersEU(t *testing.T) {
	b := newEU868()
	dlParams, err := b.GetRX1Parameters(0 /* only frequency is used in EU lookup */, 868.1, 3, 3)
	if err != nil {
		t.Error(err)
	}
	if dlParams.DataRate != 0 {
		t.Errorf("Unexpected data rate : %d", dlParams.DataRate)
	}
	if dlParams.Frequency != 868.1 {
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

func TestGetDataRateEU(t *testing.T) {
	b := newEU868()
	var dr uint8
	var err error

	dr, err = b.GetDataRate("SF12BW125")
	if (dr != 0) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF11BW125")
	if (dr != 1) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF10BW125")
	if (dr != 2) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF9BW125")
	if (dr != 3) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF8BW125")
	if (dr != 4) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF7BW125")
	if (dr != 5) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("SF7BW250")
	if (dr != 6) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}
	dr, err = b.GetDataRate("FSKBW500")
	if (dr != 7) || (err != nil) {
		t.Errorf("Unexpected data rate or error in lookup: %d. Error: %v", dr, err)
	}

	dr, err = b.GetDataRate("XYZZY")
	if err == nil {
		t.Error("Expected lookup of XYZZY to fail")
	}
}
