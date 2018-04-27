package server

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
	"encoding/hex"

	"github.com/ExploratoryEngineering/congress/model"
)

// transport implements the actual transport
type transport interface {
	// open opens the connection to the destination. If it fails it
	// will return immediately and return false. There will be several retries
	// if the connection fails. This will be called only on startup.
	open(logger *MemoryLogger) bool
	// close drops the connection and releases all resources used by
	// the connection.
	close(logger *MemoryLogger)
	// send will send a message on the connection. If the connection
	// drops out it will be reopened. The implementation must reconnect if
	// possible.
	send(msg interface{}, logger *MemoryLogger) bool
}

// GetTransport tries to create the appropriate transport for the app
// configuration. It does not open the transport.
func getTransport(op *model.AppOutput) transport {
	outputType := op.Configuration.String(model.TransportTypeKey, "unknown")
	transportFactory, ok := transports[outputType]
	if !ok {
		return nil
	}
	return transportFactory(op.Configuration)
}

type transportFactory func(model.TransportConfig) transport

// Each impplementation populates this map in their own init functino
var transports = map[string]transportFactory{}

// DeviceData is a wrapper for the model.DeviceData struct. This is used both
// in the ../data endpoints and via websockets.
//
// TODO: Merge with web socket data structure. This requires a fair bit of
//    moving around since the PayloadMessage type must be moved to its own
//    package to avoid circular dependencies.
type deviceData struct {
	DevAddr    string  `json:"devAddr"`
	Timestamp  int64   `json:"timestamp"`
	Data       string  `json:"data"`
	AppEUI     string  `json:"appEUI"`
	DeviceEUI  string  `json:"deviceEUI"`
	RSSI       int32   `json:"rssi"`
	SNR        float32 `json:"snr"`
	Frequency  float32 `json:"frequency"`
	GatewayEUI string  `json:"gatewayEUI"`
	DataRate   string  `json:"dataRate"`
}

// NewDeviceDataFromPayloadMessage converts a payload message into a DeviceData
// struct
//
// TODO: Merge with code in websocket handler. Requires PayloadMessage to be
// moved into its own package.
func newDeviceDataFromPayloadMessage(message *PayloadMessage) *deviceData {
	return &deviceData{
		DevAddr:    message.Device.DevAddr.String(),
		Timestamp:  message.FrameContext.GatewayContext.ReceivedAt.Unix(),
		Data:       hex.EncodeToString(message.Payload),
		AppEUI:     message.Application.AppEUI.String(),
		DeviceEUI:  message.Device.DeviceEUI.String(),
		RSSI:       message.FrameContext.GatewayContext.Radio.RSSI,
		SNR:        message.FrameContext.GatewayContext.Radio.SNR,
		Frequency:  message.FrameContext.GatewayContext.Radio.Frequency,
		DataRate:   message.FrameContext.GatewayContext.Radio.DataRate,
		GatewayEUI: message.FrameContext.GatewayContext.Gateway.GatewayEUI.String(),
	}
}
