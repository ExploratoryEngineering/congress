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
// BUG(hjg) No LinkAdrReq support yet.
// BUG(hjg) No CFList support yet.
// BUG(hjg) No NewChannelReq support yet.

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ModulationType is the message type
type ModulationType uint8

const (
	// LoRa is LoRa modulation
	LoRa ModulationType = 0
	// FSK is FSK modulation
	FSK ModulationType = 1
)

// RXWindowType is the RX window type
type RXWindowType uint8

const (
	// RX1 is receive window 1
	RX1 RXWindowType = 0
	// RX2 is receive window 2
	RX2 RXWindowType = 1
)

// FrequencyBandType is the frequency band type
type FrequencyBandType uint8

const (
	// EU868Band is the EU 863-870MHz ISM Band
	EU868Band FrequencyBandType = iota
	// US915Band is the US 902-928MHz ISM Band
	US915Band
	// CN780Band is the China 779-787MHz ISM Band
	CN780Band
	// EU433Band is the EU 433MHz ISM Band
	EU433Band
)

// Encoding holds the data rate specific spread factor, frequency and bit rate parameters
type Encoding struct {
	// Modulation is LoRa or FSK
	Modulation ModulationType
	// Spread factor (imho not described very well in the LoRa specification.)
	SpreadFactor uint8
	// BitRate is in bit/s
	BitRate uint32
	// Bandwidth applies to LoRa modulation only
	Bandwidth uint32
}

// MaximumPayloadSize defines max payload size
type MaximumPayloadSize struct {
	// M is max payload length if FOpts is present.
	M uint8
	// N is max payload length if FOpts is not present.
	N uint8
}

// WithFOpts return max payload if FOpts is present.
func (m *MaximumPayloadSize) WithFOpts() uint8 {
	return m.N
}

// WithoutFOpts return max payload if FOpts is not present.
func (m *MaximumPayloadSize) WithoutFOpts() uint8 {
	return m.M
}

// DownlinkParameters contains datarate and frequency
type DownlinkParameters struct {
	// DataRate used for downlink
	DataRate uint8
	// Frequency used for downlink
	Frequency float32
}

// Configuration represents frequency band specific LoRa parameters.
type Configuration struct {
	// ReceiveDelay1 is the delay (in seconds) before the first receive window opens after an uplink transmission [3.3.1].
	ReceiveDelay1 uint8
	// ReceiveDelay1 is the delay (in seconds) before the second receive window opens after an uplink transmission [3.3.2].
	ReceiveDelay2 uint8
	// JoinAccepDelay1 is the delay (in seconds) before the first window that the server can respond to a join-request message
	// with a join-accept message [6.2.5].
	JoinAccepDelay1 uint8
	// JoinAccepDelay2 is the delay (in seconds) before the first window that the server can respond to a join-request message
	// with a join-accept message [6.2.5].
	JoinAccepDelay2 uint8
	// MaxFCntGap is compared to FCnt. If the difference i greater than MaxFCntGap, then too many frames ha been lost,
	// and subsequent frames will be discarded [4.3.1.5].
	MaxFCntGap uint32
	// AdrAckLimit is used by the device to validate that the network still received uplink frames [4.3.3.1].
	AdrAckLimit uint32
	// AdrAckLimit is used by the device to validate that the network still received uplink frames [4.3.3.1].
	AdrAckDelay uint32
	// DefaultTXPower is the default radiated transmit output power in dBm [[6.25, band sub-chapters in 7]].
	DefaultTxPower uint8
	// SupportsJoinAcceptCFList indicates if the band support the optional list of channel freequencies for
	// the network the end-device is joining [Band sub-chapters in 7].
	SupportsJoinAcceptCFList bool
	// RX2Frequency is the default band frequency for the second receive window [Band sub-chapters in 7].
	RX2Frequency float32
	// RX2DataRate is the default data rate for the second receive window [Band sub-chapters in 7].
	RX2DataRate uint8

	MandatoryEndDeviceChannels []float32
	JoinReqChannels            []float32
	DownLinkFrequencies        []float32
}

//AckTimeout is the max delay limit (in seconds after the second receive window) for when then the network can send a frame with the
// ACK bit set in response to a ConfirmedData message [18.1].
func (c *Configuration) AckTimeout() int {
	return rand.Intn(3) + 1
}

// FrequencyPlan is the interface for band specific parameters
type FrequencyPlan interface {
	// Configuration returns band frequency parameters
	Configuration() *Configuration
	// Encoding returns band frequency encoding parameters
	Encoding(dataRate uint8) (Encoding, error)
	// MaximumPayload return a maximum payload size, given a data rate.
	// This implementation uses the repeater compatible definition in the LoRaWAN specification.
	MaximumPayload(dataRate string) (MaximumPayloadSize, error)
	// TxPower returns power in dBm for theb and, given a TXPower key
	TxPower(power uint8) (int8, error)
	// DownlinkDataRate returns the downlink data rate, given the upstream data rate and RX1DROffset

	// GetRX1Parameters returns datarate and frequency for downlink in receive window 1, given upstream data rate and RX1DROffset
	GetRX1Parameters(channel uint8, upstreamFrequency float32, upstreamDataRate uint8, RX1DROffset uint8) (DownlinkParameters, error)
	// GetRX2Parameters returns datarate and frequency for downlink in receive window 2.
	GetRX2Parameters() DownlinkParameters
	// GetDataRate returns data rate, given gateway representation of configuration
	GetDataRate(configuration string) (uint8, error)
	// Name returns frequencey band name.
	Name() string
}

// NewBand creates a new band configuration
func NewBand(band FrequencyBandType) (FrequencyPlan, error) {
	switch band {
	case EU868Band:
		return newEU868(), nil
	case US915Band:
		return newUS902(), nil
	default:
		return nil, fmt.Errorf("unknown band: %v. Valid arguments are: EU868, US915 (CN780 and EU433 are not implemented yet)", band)
	}
}
