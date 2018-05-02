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
import "fmt"

// EU868 represents configuration and frequency plan for the EU 863-870MHz ISM Band.
type EU868 struct {
	configuration       Configuration
	DownstreamDataRates [][]uint8
}

func newEU868() EU868 {
	return EU868{
		configuration: Configuration{
			ReceiveDelay1:            1,       // [7.1.8]
			ReceiveDelay2:            2,       // ReceiveDelay1 + 1 according to [7.1.8]
			JoinAccepDelay1:          5,       // [7.1.8]
			JoinAccepDelay2:          6,       // [7.1.8]
			MaxFCntGap:               16384,   // [7.1.8]
			AdrAckLimit:              64,      // [7.1.8]
			AdrAckDelay:              32,      // [7.1.8]
			DefaultTxPower:           14,      // [7.1.7]
			SupportsJoinAcceptCFList: true,    // [7.1.4]
			RX2Frequency:             869.525, // [7.1.7]
			RX2DataRate:              0,       // [7.1.8]
			MandatoryEndDeviceChannels: []float32{
				868.1,
				868.3,
				868.5}, // [2.1.2/Regional Parameters]
			JoinReqChannels: []float32{
				868.1,
				868.3,
				868.5}, // [2.1.2/Regional Parameters]
		},
		DownstreamDataRates: [][]uint8{
			{0, 0, 0, 0, 0, 0},
			{1, 0, 0, 0, 0, 0},
			{2, 1, 0, 0, 0, 0},
			{3, 2, 1, 0, 0, 0},
			{4, 3, 2, 1, 0, 0},
			{5, 4, 3, 2, 1, 0},
			{6, 5, 4, 3, 2, 1},
			{7, 6, 5, 4, 3, 2},
		},
	}
}

// Name returns frequency band name.
func (b EU868) Name() string {
	return "EU 863-870MHz ISM Band"
}

// Configuration returns parameters for the EU 863-870MHz ISM Band.
func (b EU868) Configuration() *Configuration {
	return &b.configuration
}

// TxPower returns power in dBm for the EU 863-870MHz ISM Band, given a TXPower key // [7.1.3]
func (b EU868) TxPower(power uint8) (int8, error) {
	switch power {
	case 0:
		return 20, nil
	case 1:
		return 14, nil
	case 2:
		return 11, nil
	case 3:
		return 8, nil
	case 4:
		return 5, nil
	case 5:
		return 2, nil
	default:
		return 0, fmt.Errorf("invalid power: %d", power)
	}
}

// Encoding returns a description of modulation, spread factor and bit rate for the EU 863-870MHz ISM Band, given a data rate. [7.1.3]
func (b EU868) Encoding(dataRate uint8) (Encoding, error) {
	switch dataRate {
	case 0:
		return Encoding{Modulation: LoRa, SpreadFactor: 12, Bandwidth: 125, BitRate: 250}, nil
	case 1:
		return Encoding{Modulation: LoRa, SpreadFactor: 11, Bandwidth: 125, BitRate: 440}, nil
	case 2:
		return Encoding{Modulation: LoRa, SpreadFactor: 10, Bandwidth: 125, BitRate: 980}, nil
	case 3:
		return Encoding{Modulation: LoRa, SpreadFactor: 9, Bandwidth: 125, BitRate: 1760}, nil
	case 4:
		return Encoding{Modulation: LoRa, SpreadFactor: 8, Bandwidth: 125, BitRate: 3125}, nil
	case 5:
		return Encoding{Modulation: LoRa, SpreadFactor: 7, Bandwidth: 125, BitRate: 5470}, nil
	case 6:
		return Encoding{Modulation: LoRa, SpreadFactor: 7, Bandwidth: 250, BitRate: 11000}, nil
	case 7:
		return Encoding{Modulation: FSK, BitRate: 50000}, nil
	default:
		return Encoding{}, fmt.Errorf("unable to look up encoding. Invalid data rate :%d", dataRate)
	}
}

// MaximumPayload return a maximum payload size, given a data rate.
// This implementation uses the repeater compatible definition in the LoRaWAN specification. [7.1.6]
func (b EU868) MaximumPayload(dataRate string) (MaximumPayloadSize, error) {
	dr, err := b.GetDataRate(dataRate)
	if err != nil {
		return MaximumPayloadSize{}, err
	}
	switch dr {
	case 0:
		return MaximumPayloadSize{M: 59, N: 51}, nil
	case 1:
		return MaximumPayloadSize{M: 59, N: 51}, nil
	case 2:
		return MaximumPayloadSize{M: 59, N: 51}, nil
	case 3:
		return MaximumPayloadSize{M: 123, N: 115}, nil
	case 4:
		return MaximumPayloadSize{M: 230, N: 222}, nil
	case 5:
		return MaximumPayloadSize{M: 230, N: 222}, nil
	case 6:
		return MaximumPayloadSize{M: 230, N: 222}, nil
	case 7:
		return MaximumPayloadSize{M: 230, N: 222}, nil
	default:
		return MaximumPayloadSize{}, fmt.Errorf("unable to look up maximum payload. Invalid data rate :%s", dataRate)
	}
}

// GetRX1Parameters returns datarate and frequency for downlink in receive window 1, given upstream data rate and RX1DROffset
func (b EU868) GetRX1Parameters(channel uint8, upstreamFrequency float32, upstreamDataRate uint8, RX1DROffset uint8) (DownlinkParameters, error) {
	datarate, err := b.downlinkDataRate(upstreamDataRate, RX1DROffset)
	return DownlinkParameters{DataRate: datarate, Frequency: upstreamFrequency}, err
}

// GetRX2Parameters returns datarate and frequency for downlink in receive window 2.
func (b EU868) GetRX2Parameters() DownlinkParameters {
	return DownlinkParameters{DataRate: b.configuration.RX2DataRate, Frequency: b.configuration.RX2Frequency}
}

// DownlinkDataRate returns the downlink data rate, given the upstream data rate and RX1DROffset [7.1.7]
func (b EU868) downlinkDataRate(upstreamDataRate uint8, RX1DROffset uint8) (uint8, error) {
	if upstreamDataRate > 7 {
		return 0, fmt.Errorf("invalid data rate parameter: %d. Data rate has to be in the interval [0, 7]", upstreamDataRate)
	}
	if RX1DROffset > 5 {
		return 0, fmt.Errorf("invalid RX1DROffset parameter: %d. RX1DROffset has to be in the interval [0, 5]", RX1DROffset)
	}

	return b.DownstreamDataRates[upstreamDataRate][RX1DROffset], nil
}

// GetDataRate returns data rate, given gateway representation of configuration
func (b EU868) GetDataRate(configuration string) (uint8, error) {
	switch configuration {
	case "SF12BW125":
		return 0, nil
	case "SF11BW125":
		return 1, nil
	case "SF10BW125":
		return 2, nil
	case "SF9BW125":
		return 3, nil
	case "SF8BW125":
		return 4, nil
	case "SF7BW125":
		return 5, nil
	case "SF7BW250":
		return 6, nil
	case "FSKBW500":
		return 7, nil
	default:
		return 0, fmt.Errorf("unable to convert configuration '%s' into data rate", configuration)
	}
}
