package model

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
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/logging"
)

// Data model for LoRaWAN network. The data model is used in parts of the decoding

// User is the user owning/administering applications. Authentication and
// authorization are performed by external systems.
type User struct {
	ID    UserID
	Name  string
	Email string
}

// Application represents a LoRa application instance.
type Application struct {
	AppEUI protocol.EUI // Application EUI
	Tags
}

// Equals returns true if the other application has identical fields. Just like
// ...Equals
func (a *Application) Equals(other Application) bool {
	return a.AppEUI == other.AppEUI &&
		a.Tags.Equals(other.Tags)
}

// NewApplication creates a new application instance
func NewApplication() Application {
	return Application{
		Tags: NewTags(),
	}
}

// GenerateAppNonce generates a new AppNonce, three random bytes that will be used
// to generate new devices.
func (a *Application) GenerateAppNonce() ([3]byte, error) {
	var nonce [3]byte
	_, err := rand.Read(nonce[:])
	return nonce, err
}

// DeviceState represents the device state
type DeviceState uint8

// Types of devices. A device can either be OTAA, ABP or disabled.
const (
	OverTheAirDevice   DeviceState = 1
	PersonalizedDevice DeviceState = 8 // Note: 8 is for backwards compatibility
	DisabledDevice     DeviceState = 0
)

// String converts the device state into a human-readable string representation.
func (d DeviceState) String() string {
	switch d {
	case OverTheAirDevice:
		return "OTAA"
	case PersonalizedDevice:
		return "ABP"
	case DisabledDevice:
		return "Disabled"
	default:
		logging.Warning("Unknown device state: %d", d)
		return "Disabled"
	}
}

// DeviceStateFromString converts a string representation of DeviceState into
// a DeviceState value. Unknown strings returns the DisabledDevice state.
// Conversion is not case sensitive. White space is trimmed.
func DeviceStateFromString(str string) (DeviceState, error) {
	switch strings.TrimSpace(strings.ToUpper(str)) {
	case "OTAA":
		return OverTheAirDevice, nil
	case "ABP":
		return PersonalizedDevice, nil
	case "DISABLED":
		return DisabledDevice, nil
	default:
		return DisabledDevice, fmt.Errorf("unknown device state: %s", str)
	}
}

// Device represents a device. Devices are associated with one and only one Application
type Device struct {
	DeviceEUI       protocol.EUI     // EUI for device
	DevAddr         protocol.DevAddr // Device address
	AppKey          protocol.AESKey  // AES key for application
	AppSKey         protocol.AESKey  // Application session key
	NwkSKey         protocol.AESKey  // Network session key
	AppEUI          protocol.EUI     // The application associated with the device. Set by storage backend
	State           DeviceState      // Current state of the device
	FCntUp          uint16           // Frame counter up (from device)
	FCntDn          uint16           // Frame counter down (to device)
	RelaxedCounter  bool             // Relaxed frame count checks
	DevNonceHistory []uint16         // Log of DevNonces sent from the device
	KeyWarning      bool             // Duplicate key warning flag
	Tags
}

// NewDevice creates a new device
func NewDevice() Device {
	return Device{Tags: NewTags()}
}

// GetRX1Window returns the 1st receive window for the device
// BUG(stlaehd): Returns constant. Should be set based on device settings and frequency plan.
func (d *Device) GetRX1Window() time.Duration {
	return time.Second * 1
}

// GetRX2Window returns the 2nd receive window for the device
// BUG(stalehd): Returns a constant. Should be set based on frequency plan (EU, US, CN)
func (d *Device) GetRX2Window() time.Duration {
	return time.Second * 2
}

// HasDevNonce returns true if the specified nonce exists in the nonce history
func (d *Device) HasDevNonce(devNonce uint16) bool {
	for _, v := range d.DevNonceHistory {
		if v == devNonce {
			return true
		}
	}
	return false
}

// DeviceData contains a single transmission from an end-device.
type DeviceData struct {
	DeviceEUI  protocol.EUI     // Device address used
	Timestamp  int64            // Timestamp for message. Data type might change.
	Data       []byte           // The data the end-device sent
	GatewayEUI protocol.EUI     // The gateway the message was received from.
	RSSI       int32            // Radio stats; RSSI
	SNR        float32          // Radio; SNR
	Frequency  float32          // Radio; Frequency
	DataRate   string           // Data rate (ie "SF7BW125" or similar)
	DevAddr    protocol.DevAddr // The reported DevAddr (at the time)
}

// Equals compares two DeviceData instances
func (d *DeviceData) Equals(other DeviceData) bool {
	return bytes.Compare(d.Data, other.Data) == 0 &&
		d.DeviceEUI.String() == other.DeviceEUI.String() &&
		d.Timestamp == other.Timestamp &&
		d.GatewayEUI.String() == other.GatewayEUI.String() &&
		d.RSSI == other.RSSI &&
		d.SNR == other.SNR &&
		d.Frequency == other.Frequency &&
		d.DataRate == other.DataRate &&
		d.DevAddr == other.DevAddr

}

// PublicGatewayInfo contains public information for gateways. This information
// is available to all users, even the ones not owning a gateway.
type PublicGatewayInfo struct {
	EUI       string  // EUI of gateway.
	Latitude  float32 // Latitude, in decimal degrees, positive N <-90-90>
	Longitude float32 // Longitude, in decimal degrees, positive E [-180-180>
	Altitude  float32 // Altitude, meters
}

// Gateway represents - you guessed it - a gateway.
type Gateway struct {
	GatewayEUI protocol.EUI // EUI of gateway.
	IP         net.IP       // IP address of gateway. This might not be fixed.
	StrictIP   bool         // Strict IP address check
	Latitude   float32      // Latitude, in decimal degrees, positive N <-90-90>
	Longitude  float32      // Longitude, in decimal degrees, positive E [-180-180>
	Altitude   float32      // Altitude, meters
	Tags
}

// NewGateway creates a new gateway
func NewGateway() Gateway {
	return Gateway{Tags: NewTags()}
}

// Equals checks gateways for equality
func (g *Gateway) Equals(other Gateway) bool {
	return g.Altitude == other.Altitude &&
		g.GatewayEUI.String() == other.GatewayEUI.String() &&
		g.IP.Equal(other.IP) &&
		g.Latitude == other.Latitude &&
		g.Longitude == other.Longitude &&
		g.StrictIP == other.StrictIP &&
		g.Tags.Equals(other.Tags)
}

// APIToken represents an API token that the users can use to access the
// API. There are two basic roles -- "read" and "write", set by the
// ReadOnly flag.
//
// The resource is a HTTP URI. A token with the resource
//
//    /applications/<eui>
//
// would give access to that particular application while a token with
// the resource set to
//
//    /applications
//
// would give access to all of the applications.
// The most relaxed resource is a single `/` which gives access to
// *all* resources owned by the user.
// The user ID is the CONNECT ID user id. The token itself is a string of
// alphanumeric characters.
type APIToken struct {
	Token    string
	UserID   UserID
	Write    bool
	Resource string
	Tags
}

// NewAPIToken creates a new API token.
func NewAPIToken(userID UserID, resource string, write bool) (APIToken, error) {
	buf := make([]byte, 32)
	n, err := rand.Read(buf)
	if err == nil && n != len(buf) {
		err = fmt.Errorf("unable to generate token %d bytes long. Only got %d bytes", len(buf), n)
	}
	token := hex.EncodeToString(buf)
	return APIToken{token, userID, write, resource, NewTags()}, err
}

// Equals return true if the token is equal to ther provided token
func (t *APIToken) Equals(other APIToken) bool {
	return t.Token == other.Token && t.UserID == other.UserID &&
		t.Write == other.Write && t.Resource == other.Resource &&
		t.Tags.Equals(other.Tags)
}

// UserID is a type representing the users
type UserID string

const (
	// SystemUserID is the user ID for the "system user". It should not be used for
	// anything but internal access.
	SystemUserID UserID = "system"
	// InvalidUserID is an user ID that is invalid.
	InvalidUserID UserID = ""
)

// TransportConfigKey is the key for configuration items
type TransportConfigKey string

const (
	// TransportTypeKey is the key used to identify the output (and configuration)
	TransportTypeKey = TransportConfigKey("type")
)

// TransportConfig is a generic configuration for outputs. This is in effect a
// simple map keyed on a string.
type TransportConfig map[TransportConfigKey]interface{}

// String returns the key value as a string. If the key doesn't exist or the
// key is of a different type the default will be returned
func (t TransportConfig) String(key TransportConfigKey, def string) string {
	val, ok := t[key].(string)
	if !ok {
		return def
	}
	return val
}

// Bool returns the key value as a boolean. If the key doesn't exist or the
// key is of a different type the default will be returned
func (t TransportConfig) Bool(key TransportConfigKey, def bool) bool {
	val, ok := t[key].(bool)
	if !ok {
		return def
	}
	return val
}

// Int returns the key value as a boolean. If the key doesn't exist or the
// key is of a different type the default will be returned
func (t TransportConfig) Int(key TransportConfigKey, def int) int {
	val, ok := t[key].(float64)
	if !ok {
		return def
	}
	return int(val)
}

// AppOutput is configuration for application outputs (MQTT et al)
type AppOutput struct {
	EUI           protocol.EUI    // Unique identifier for output
	AppEUI        protocol.EUI    // The associated application
	Configuration TransportConfig // Output configuration
}

// NewTransportConfig creates a new OutputConfig from a JSON string
func NewTransportConfig(jsonString string) (TransportConfig, error) {
	ret := TransportConfig{}
	return ret, json.Unmarshal([]byte(jsonString), &ret)
}

// NewAppOutput creates a new output instance
func NewAppOutput() AppOutput {
	return AppOutput{Configuration: make(map[TransportConfigKey]interface{})}
}

// DownstreamMessageState is the state of the downstream messages
type DownstreamMessageState uint8

// States for the downstream message
const (
	UnsentState DownstreamMessageState = iota
	SentState
	AcknowledgedState
)

// DownstreamMessage is messages sent downstream (ie to devices from the server).
type DownstreamMessage struct {
	DeviceEUI protocol.EUI
	Data      string
	Port      uint8

	Ack         bool
	CreatedTime int64
	SentTime    int64
	AckTime     int64
}

// NewDownstreamMessage creates a new DownstreamMessage
func NewDownstreamMessage(deviceEUI protocol.EUI, port uint8) DownstreamMessage {
	return DownstreamMessage{deviceEUI, "", port, false, time.Now().Unix(), 0, 0}
}

// State returns the message's state based on the value of the time stamps
func (d *DownstreamMessage) State() DownstreamMessageState {
	// Sent time isn't updated => message is still pending
	if d.SentTime == 0 {
		return UnsentState
	}
	// Message isn't acknowledged but sent time is set => message is sent
	if d.AckTime == 0 {
		return SentState
	}
	// AckTime and SentTime is set => acknowledged
	return AcknowledgedState
}

// Payload returns the payload as a byte array. If there's an error decoding the
// data it will return an empty byte array
func (d *DownstreamMessage) Payload() []byte {
	ret, err := hex.DecodeString(d.Data)
	if err != nil {
		logging.Warning("Unable to decode data to be sent to device %s (data=%s). Ignoring it.", d.DeviceEUI, d.Data)
		return []byte{}
	}
	return ret
}

// IsComplete returns true if the message processing is completed. If the ack
// flag isn't set the message would only have to be sent to the device. If the
// ack flag is set the device must acknowledge the message before it is
// considered completed.
func (d *DownstreamMessage) IsComplete() bool {
	// Message haven't been sent yet
	if d.SentTime == 0 {
		return false
	}
	// Message have been sent but not acknowledged yet
	if d.Ack && d.AckTime == 0 {
		return false
	}
	return true
}
