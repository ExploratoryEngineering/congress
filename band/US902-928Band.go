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

// US902 represents configuration and frequency plan for the US 902-928MHz ISM Band.
type US902 struct {
	configuration       Configuration
	DownstreamDataRates [][]uint8
	DownstreamChannels  []float32
}

func newUS902() US902 {
	return US902{
		configuration: Configuration{
			ReceiveDelay1:            1,     // [7.2.8]
			ReceiveDelay2:            2,     // ReceiveDelay1 + 1 according to [7.2.8]
			JoinAccepDelay1:          5,     // [7.2.8]
			JoinAccepDelay2:          6,     // [7.2.8]
			MaxFCntGap:               16384, // [7.2.8]
			AdrAckLimit:              64,    // [7.2.8]
			AdrAckDelay:              32,    // [7.2.8]
			DefaultTxPower:           20,    // or a) 30 dBm for 125kHz BW (max 400ms), or b) 26 dBm for 500kHz BW [7.2.1]
			SupportsJoinAcceptCFList: false, // [7.2.4]
			RX2Frequency:             923.3, // [7.2.7]
			RX2DataRate:              8,     // [7.2.7]
		},
		DownstreamDataRates: [][]uint8{
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
		},
		DownstreamChannels: []float32{
			923.3,
			923.9,
			924.5,
			925.1,
			925.7,
			926.3,
			926.9,
			927.5,
		},
	}
}

// Name returns frequency band name.
func (b US902) Name() string {
	return "US 902-928MHz ISM Band"
}

// Configuration returns parameters for the US 902-928MHz ISM Band.
func (b US902) Configuration() *Configuration {
	return &b.configuration
}

// TxPower returns power in dBm for the US 902-928MHz ISM Band, given a TXPower  [7.2.3]
func (b US902) TxPower(power uint8) (int8, error) {
	if power > 10 {
		return 0, fmt.Errorf("invalid power : %d", power)
	}
	return int8(30 - 2*power), nil
}

// Encoding returns a description of modulation, spread factor and bit rate for the US 902-928MHz ISM Band, given a data rate. [7.2.3]
func (b US902) Encoding(dataRate uint8) (Encoding, error) {

	switch dataRate {
	case 0:
		return Encoding{Modulation: LoRa, SpreadFactor: 10, Bandwidth: 125, BitRate: 980}, nil
	case 1:
		return Encoding{Modulation: LoRa, SpreadFactor: 9, Bandwidth: 125, BitRate: 1760}, nil
	case 2:
		return Encoding{Modulation: LoRa, SpreadFactor: 8, Bandwidth: 125, BitRate: 3125}, nil
	case 3:
		return Encoding{Modulation: LoRa, SpreadFactor: 7, Bandwidth: 125, BitRate: 5470}, nil
	case 4:
		return Encoding{Modulation: LoRa, SpreadFactor: 8, Bandwidth: 500, BitRate: 12500}, nil
	case 8:
		return Encoding{Modulation: LoRa, SpreadFactor: 12, Bandwidth: 500, BitRate: 980}, nil
	case 9:
		return Encoding{Modulation: LoRa, SpreadFactor: 11, Bandwidth: 500, BitRate: 1760}, nil
	case 10:
		return Encoding{Modulation: LoRa, SpreadFactor: 10, Bandwidth: 500, BitRate: 3900}, nil
	case 11:
		return Encoding{Modulation: LoRa, SpreadFactor: 9, Bandwidth: 500, BitRate: 7000}, nil
	case 12:
		return Encoding{Modulation: LoRa, SpreadFactor: 8, Bandwidth: 500, BitRate: 12500}, nil
	case 13:
		return Encoding{Modulation: LoRa, SpreadFactor: 7, Bandwidth: 500, BitRate: 21900}, nil
	default:
		return Encoding{}, fmt.Errorf("unable to look up encoding. Invalid data rate :%d (Datarates 5-7 are RFU)", dataRate)
	}
}

// MaximumPayload return a maximum payload size, given a data rate.
// This implementation uses the repeater compatible definition in the LoRaWAN specification. [7.2.6]
func (b US902) MaximumPayload(dataRate string) (MaximumPayloadSize, error) {
	dr, err := b.GetDataRate(dataRate)
	if err != nil {
		return MaximumPayloadSize{}, err
	}
	switch dr {
	case 0:
		return MaximumPayloadSize{M: 19, N: 11}, nil
	case 1:
		return MaximumPayloadSize{M: 61, N: 53}, nil
	case 2:
		return MaximumPayloadSize{M: 137, N: 129}, nil
	case 3:
		return MaximumPayloadSize{M: 250, N: 242}, nil
	case 4:
		return MaximumPayloadSize{M: 250, N: 242}, nil
	case 8:
		return MaximumPayloadSize{M: 41, N: 33}, nil
	case 9:
		return MaximumPayloadSize{M: 117, N: 109}, nil
	case 10:
		return MaximumPayloadSize{M: 230, N: 222}, nil
	case 11:
		return MaximumPayloadSize{M: 230, N: 222}, nil
	case 12:
		return MaximumPayloadSize{M: 230, N: 222}, nil
	case 13:
		return MaximumPayloadSize{M: 230, N: 222}, nil

	default:
		return MaximumPayloadSize{}, fmt.Errorf("unable to look up maximum payload. Invalid data rate:%s (Datarates 5-7 are RFU)", dataRate)
	}
}

// GetRX1Parameters returns datarate and frequency for downlink in receive window 1, given upstream data rate and RX1DROffset
func (b US902) GetRX1Parameters(channel uint8, upstreamFrequency float32, upstreamDataRate uint8, RX1DROffset uint8) (DownlinkParameters, error) {
	datarate, err := b.downlinkDataRate(upstreamDataRate, RX1DROffset)
	return DownlinkParameters{DataRate: datarate, Frequency: b.DownstreamChannels[channel%8]}, err
}

// GetRX2Parameters returns datarate and frequency for downlink in receive window 2.
func (b US902) GetRX2Parameters() DownlinkParameters {
	return DownlinkParameters{DataRate: b.configuration.RX2DataRate, Frequency: b.configuration.RX2Frequency}
}

// DownlinkDataRate returns the downlink data rate, given the upstream data rate and RX1DROffset [7.2.7]
func (b US902) downlinkDataRate(upstreamDataRate uint8, RX1DROffset uint8) (uint8, error) {
	if upstreamDataRate > 13 || (upstreamDataRate > 4 && upstreamDataRate < 8) {
		return 0, fmt.Errorf("invalid data rate parameter: %d. Data rate has to be in the interval [0, 7]", upstreamDataRate)
	}
	if RX1DROffset > 3 {
		return 0, fmt.Errorf("invalid RX1DROffset parameter: %d. RX1DROffset has to be in the interval [0, 5]", RX1DROffset)
	}

	return b.DownstreamDataRates[upstreamDataRate][RX1DROffset], nil
}

// GetDataRate returns data rate, given gateway representation of configuration
// (DR4 is identical to DR12. Defaulting to DR4)
func (b US902) GetDataRate(configuration string) (uint8, error) {
	switch configuration {
	case "SF10BW125":
		return 0, nil
	case "SF9BW125":
		return 1, nil
	case "SF8BW125":
		return 2, nil
	case "SF7BW125":
		return 3, nil
	case "SF8BW500":
		return 4, nil
	case "SF12BW500":
		return 8, nil
	case "SF11BW500":
		return 9, nil
	case "SF10BW500":
		return 10, nil
	case "SF9BW500":
		return 11, nil
	case "SF7BW500":
		return 13, nil
	default:
		return 0, fmt.Errorf("unknown configuration: %s", configuration)
	}
}
