package restapi

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
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/logging"
)

// A set of entities to make the conversion to and from API JSON types
// less annoying.

// HyperlinkTemplate is a struct holding links and templates for resources
type hyperlinkTemplate struct {
	Links     map[string]string `json:"links"`
	Templates map[string]string `json:"templates"`
}

func appDeviceTemplates() map[string]string {
	return map[string]string{
		"application-collection": "/applications",
		"application-data":       "/applications/{aeui}/data{?limit&since}",
		"application-stream":     "/applications/{aeui}/stream",
		"device-collection":      "/applications/{aeui}/devices",
		"device-data":            "/applications/{aeui}/devices/{deui}/data{?limit&since}",
		"gateways":               "/gateways",
		"gateway-info":           "/gateways/{geui}",
	}
}

// apiApplication is the entity used by the REST API for applications
type apiApplication struct {
	ApplicationEUI string `json:"applicationEUI"`
	eui            protocol.EUI
	Tags           map[string]string `json:"tags"`
}

// ApplicationList is the list of applications presented by the REST API
type applicationList struct {
	Applications []apiApplication  `json:"applications"`
	Templates    map[string]string `json:"templates"`
}

// NewApplicationList creates a new application list
func newApplicationList() applicationList {
	return applicationList{
		Applications: make([]apiApplication, 0),
		Templates:    appDeviceTemplates(),
	}
}

// NewAppFromModel creates a new application from a model.Application instance
func newAppFromModel(app model.Application) apiApplication {
	return apiApplication{
		ApplicationEUI: app.AppEUI.String(),
		eui:            app.AppEUI,
		Tags:           app.Tags.Tags(),
	}
}

// ToModel converts the API application into a model.Application entity
func (a *apiApplication) ToModel() model.Application {
	tags, err := model.NewTagsFromMap(a.Tags)
	if err != nil {
		logging.Info("Unable to convert tags from API entity to model: %v", err)
		// Replace with empty tags
		tmp := model.NewTags()
		tags = &tmp
	}
	return model.Application{
		AppEUI: a.eui,
		Tags:   *tags,
	}
}

func (a *apiApplication) equals(other apiApplication) bool {
	return a.ApplicationEUI == other.ApplicationEUI &&
		reflect.DeepEqual(a.Tags, other.Tags)
}

// Types of devices; ABP/OTAA
const (
	deviceTypeABP  string = "ABP"
	deviceTypeOTAA string = "OTAA"
)

// APIDevice is the REST API type used for devices
type apiDevice struct {
	DeviceEUI      string `json:"deviceEUI"`
	DevAddr        string `json:"devAddr"`
	AppKey         string `json:"appKey"`
	AppSKey        string `json:"appSKey"`
	NwkSKey        string `json:"nwkSKey"`
	FCntUp         uint16 `json:"fCntUp"`
	FCntDn         uint16 `json:"fCntDn"`
	RelaxedCounter bool   `json:"relaxedCounter"`
	DeviceType     string `json:"deviceType"`
	KeyWarning     bool   `json:"keyWarning"`
	eui            protocol.EUI
	da             protocol.DevAddr
	akey           protocol.AESKey
	askey          protocol.AESKey
	nskey          protocol.AESKey
	Tags           map[string]string `json:"tags"`
}

// NewDeviceFromModel creates an APIDevice instance from a model.Device instance.
func newDeviceFromModel(device *model.Device) apiDevice {
	var state = deviceTypeOTAA
	if device.State == model.PersonalizedDevice {
		state = deviceTypeABP
	}
	return apiDevice{
		DeviceEUI:      device.DeviceEUI.String(),
		eui:            device.DeviceEUI,
		DevAddr:        device.DevAddr.String(),
		da:             device.DevAddr,
		akey:           device.AppKey,
		AppKey:         device.AppKey.String(),
		AppSKey:        device.AppSKey.String(),
		askey:          device.AppSKey,
		NwkSKey:        device.NwkSKey.String(),
		nskey:          device.NwkSKey,
		FCntDn:         device.FCntDn,
		FCntUp:         device.FCntUp,
		RelaxedCounter: device.RelaxedCounter,
		DeviceType:     state,
		KeyWarning:     device.KeyWarning,
		Tags:           device.Tags.Tags(),
	}
}

// ToModel converts the instance into model.Device instance
func (d *apiDevice) ToModel(appEUI protocol.EUI) model.Device {
	var state = model.OverTheAirDevice
	if strings.ToUpper(d.DeviceType) == deviceTypeABP {
		state = model.PersonalizedDevice
	}

	tags, err := model.NewTagsFromMap(d.Tags)
	if err != nil {
		logging.Warning("Unable to convert api device tags to model tags: %v. Ignoring", err)
	}
	return model.Device{
		DeviceEUI:      d.eui,
		DevAddr:        d.da,
		AppKey:         d.akey,
		AppSKey:        d.askey,
		NwkSKey:        d.nskey,
		AppEUI:         appEUI,
		State:          state,
		FCntDn:         d.FCntDn,
		FCntUp:         d.FCntUp,
		RelaxedCounter: d.RelaxedCounter,
		KeyWarning:     d.KeyWarning,
		Tags:           *tags,
	}
}

// DeviceList is the list of devices
type deviceList struct {
	Devices   []apiDevice       `json:"devices"`
	Templates map[string]string `json:"templates"`
}

// NewDeviceList creates a new device list
func newDeviceList() deviceList {
	return deviceList{
		Devices:   make([]apiDevice, 0),
		Templates: appDeviceTemplates(),
	}
}

// apiGateway is used to convert to and from JSON
type apiGateway struct {
	GatewayEUI string            `json:"gatewayEUI"`
	IP         string            `json:"ip"`
	StrictIP   bool              `json:"strictIP"`
	Latitude   float32           `json:"latitude"`
	Longitude  float32           `json:"longitude"`
	Altitude   float32           `json:"altitude"`
	Tags       map[string]string `json:"tags"`
	eui        protocol.EUI
	ipaddr     net.IP
}

// ToModel converts an APIGateway instance to a model.Gateway
func (g *apiGateway) ToModel() model.Gateway {
	eui, _ := protocol.EUIFromString(g.GatewayEUI)
	tags, err := model.NewTagsFromMap(g.Tags)
	if err != nil {
		logging.Warning("Unable to convert API tags to tags struct: %v", err)
	}
	return model.Gateway{
		GatewayEUI: eui,
		IP:         net.ParseIP(g.IP),
		StrictIP:   g.StrictIP,
		Latitude:   g.Latitude,
		Longitude:  g.Longitude,
		Altitude:   g.Altitude,
		Tags:       *tags,
	}
}

// NewGatewayFromModel creates a new APIGateway instance from a model.Gateway instance
func newGatewayFromModel(gateway model.Gateway) apiGateway {
	return apiGateway{
		GatewayEUI: gateway.GatewayEUI.String(),
		IP:         gateway.IP.String(),
		StrictIP:   gateway.StrictIP,
		Latitude:   gateway.Latitude,
		Longitude:  gateway.Longitude,
		Altitude:   gateway.Altitude,
		Tags:       gateway.Tags.Tags(),
	}
}

// GatewayList is the list of gateways
type gatewayList struct {
	Gateways  []apiGateway      `json:"gateways"`
	Templates map[string]string `json:"templates"`
}

// NewGatewayList returns an unpopulated list of gateways
func newGatewayList() gatewayList {
	return gatewayList{
		Gateways: make([]apiGateway, 0),
		Templates: map[string]string{
			"gateway-list": "/gateways",
			"gateway-info": "/gateways/{geui}",
		},
	}
}

type apiPublicGateway struct {
	EUI       string  `json:"gatewayEui"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
	Altitude  float32 `json:"altitude"`
}

type apiPublicGatewayList struct {
	Gateways []apiPublicGateway `json:"gateways"`
}

func newPublicGatewayFromModel(gateway model.PublicGatewayInfo) apiPublicGateway {
	return apiPublicGateway{
		EUI:       gateway.EUI,
		Latitude:  gateway.Latitude,
		Longitude: gateway.Longitude,
		Altitude:  gateway.Altitude,
	}
}

// ToUnixMillis converts a nanosecond timestamp into a millisecond timestamp.
// the general assumption is that time.Nanosecond = 1 (which it is)
func ToUnixMillis(unixNanos int64) int64 {
	return unixNanos / int64(time.Millisecond)
}

// FromUnixMillis converts a millisecond timestamp into nanosecond timestamp. Note
// that this assumes that time.Nanosecond = 1 (which it is)
func FromUnixMillis(unixMillis int64) int64 {
	return unixMillis * int64(time.Millisecond)
}

// APIDeviceData is a wrapper for the model.DeviceData struct. This is used both
// in the ../data endpoints and via websockets.
type apiDeviceData struct {
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

// NewDeviceDataFromModel returns an user-friendly version of the DeviceData struct
func newDeviceDataFromModel(data model.DeviceData, appEUI protocol.EUI) apiDeviceData {
	return apiDeviceData{
		AppEUI:     appEUI.String(),
		DeviceEUI:  data.DeviceEUI.String(),
		Timestamp:  ToUnixMillis(data.Timestamp),
		DevAddr:    data.DevAddr.String(),
		Data:       hex.EncodeToString(data.Data),
		GatewayEUI: data.GatewayEUI.String(),
		RSSI:       data.RSSI,
		SNR:        data.SNR,
		Frequency:  data.Frequency,
		DataRate:   data.DataRate,
	}
}

// APIDataList is the list of data packets from devices
type apiDataList struct {
	Messages []apiDeviceData `json:"messages"`
}

// NewAPIDataList returns a new APIDataList instance
func newAPIDataList() apiDataList {
	return apiDataList{Messages: make([]apiDeviceData, 0)}
}

// apiDownstreamMessage is a message that will be sent to a device. The message
// is very similar to the existing model entity but for consistency's sake
// it will be treated like other entities.
type apiDownstreamMessage struct {
	DeviceEUI   string `json:"deviceEUI"`
	Data        string `json:"data"`
	Port        uint8  `json:"port"`
	Ack         bool   `json:"ack"`
	SentTime    int64  `json:"sentTime"`
	CreatedTime int64  `json:"createdTime"`
	AckTime     int64  `json:"ackTime"`
	State       string `json:"state"`
}

// ToModel converts the end-user message into model.DownstreamMessage
func (m *apiDownstreamMessage) ToModel() (model.DownstreamMessage, error) {
	deviceEUI, err := protocol.EUIFromString(m.DeviceEUI)
	if err != nil {
		return model.DownstreamMessage{}, err
	}
	return model.DownstreamMessage{
		DeviceEUI:   deviceEUI,
		Data:        m.Data,
		Port:        m.Port,
		Ack:         m.Ack,
		SentTime:    m.SentTime,
		CreatedTime: m.CreatedTime,
		AckTime:     m.AckTime,
	}, nil
}

func newDownstreamMessageFromModel(msg model.DownstreamMessage) apiDownstreamMessage {
	var state string
	switch msg.State() {
	case model.UnsentState:
		state = "UNSENT"
	case model.SentState:
		state = "SENT"
	case model.AcknowledgedState:
		state = "ACKNOWLEDGED"
	}
	return apiDownstreamMessage{
		DeviceEUI:   msg.DeviceEUI.String(),
		Data:        msg.Data,
		Port:        msg.Port,
		Ack:         msg.Ack,
		SentTime:    msg.SentTime,
		CreatedTime: msg.CreatedTime,
		AckTime:     msg.AckTime,
		State:       state,
	}
}

// apiToken is a wrapper type for model.APIToken that omits some of the fields
// in the original struct. The APIToken struct could have been used as is but
// this is consistent with the other types in the model package.
type apiToken struct {
	Token    string            `json:"token"`
	Write    bool              `json:"write"`
	Resource string            `json:"resource"`
	Tags     map[string]string `json:"tags"`
}

func newTokenFromModel(existing model.APIToken) apiToken {
	return apiToken{
		Token:    existing.Token,
		Write:    existing.Write,
		Resource: existing.Resource,
		Tags:     existing.Tags.Tags(),
	}
}

// This is a list of tokens
type apiTokenList struct {
	Tokens []apiToken `json:"tokens"`
}

// Outputs are data outputs from applications. There's just one output
// type ATM.

// apiOutputList is a list of outputs to the client
type apiAppOutputList struct {
	List []apiAppOutput `json:"outputs"`
}

// apiLog is a list of log entries from an output
type apiAppOutputLog struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// apiOutput is an output presented to the client
type apiAppOutput struct {
	EUI    string                `json:"eui"`
	AppEUI string                `json:"appEUI"`
	Config model.TransportConfig `json:"config"`
	Log    []apiAppOutputLog     `json:"logs,omitempty"`
	Status string                `json:"status"`
}

// newOutputFromModel convers a model output to a client-friendly output
func newOutputFromModel(src model.AppOutput, log *server.MemoryLogger, status string) apiAppOutput {
	var logMessages []apiAppOutputLog
	for _, v := range log.Entries {
		if v.IsValid() {
			logMessages = append(logMessages, apiAppOutputLog{v.TimeString(), v.Message})
		}
	}
	return apiAppOutput{
		EUI:    src.EUI.String(),
		AppEUI: src.AppEUI.String(),
		Config: src.Configuration,
		Log:    logMessages,
		Status: status,
	}
}

// ToModel converts the apiOutput into a model equivalent
func (a *apiAppOutput) ToModel() (model.AppOutput, error) {
	ret := model.NewAppOutput()
	var err error
	if ret.EUI, err = protocol.EUIFromString(a.EUI); err != nil {
		return ret, err
	}
	if ret.AppEUI, err = protocol.EUIFromString(a.AppEUI); err != nil {
		return ret, err
	}
	if a.Config != nil {
		ret.Configuration = a.Config
	}
	return ret, nil
}
