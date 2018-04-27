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
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/ExploratoryEngineering/congress/monitoring"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

func (s *Server) deviceList(w http.ResponseWriter, r *http.Request, appEUI protocol.EUI) {
	devices, err := s.context.Storage.Device.GetByApplicationEUI(appEUI)
	if err != nil {
		logging.Warning("Unable to read device list for application %s: %v.", appEUI, err)
		http.Error(w, "Server error", http.StatusNotFound)
	}
	deviceList := newDeviceList()
	for device := range devices {
		deviceList.Devices = append(deviceList.Devices, newDeviceFromModel(&device))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(deviceList); err != nil {
		logging.Warning("Unable to marshal device list for application %s: %v", appEUI, err)
	}
}

func (s *Server) createDevice(w http.ResponseWriter, r *http.Request, applicationEUI protocol.EUI) {
	// POST methods contains a single JSON struct in the body. Only one device instance is processed.
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logging.Warning("Unable to read request body for device POST: %v", err)
		http.Error(w, "Unable to read request body", http.StatusInternalServerError)
		return
	}
	device := apiDevice{}
	if err = json.Unmarshal(buf, &device); err != nil {
		logging.Info("Unable to unmarshal JSON for device: %v", err)
		http.Error(w, "Can't grok JSON", http.StatusBadRequest)
		return
	}

	deviceType := model.OverTheAirDevice
	if strings.ToUpper(device.DeviceType) == deviceTypeABP {
		deviceType = model.PersonalizedDevice
	}

	var overrideEUI, overrideDevAddr, overrideAppKey, overrideAppSKey, overrideNwkSKey bool
	if device.DeviceEUI != "" {
		device.eui, err = protocol.EUIFromString(device.DeviceEUI)
		if err != nil {
			http.Error(w, "Invalid device EUI specified", http.StatusBadRequest)
			return
		}
		overrideEUI = true
	}

	if !overrideEUI {
		device.eui, err = s.context.KeyGenerator.NewDeviceEUI()
		if err != nil {
			logging.Warning("Unable to generate EUI for device: %v", err)
			http.Error(w, "Unable to generate EUI for device", http.StatusInternalServerError)
			return
		}
		device.DeviceEUI = device.eui.String()
	}

	if device.DevAddr != "" {
		if deviceType != model.PersonalizedDevice {
			http.Error(w, "DevAddr can only be specified for ABP devices", http.StatusBadRequest)
			return
		}
		if device.da, err = protocol.DevAddrFromString(device.DevAddr); err != nil {
			http.Error(w, "Invalid DevAddr", http.StatusBadRequest)
			return
		}
		device.DevAddr = device.da.String()
		overrideDevAddr = true
	}

	if !overrideDevAddr {
		device.da = protocol.NewDevAddr()
		device.DevAddr = device.da.String()
	}

	if device.AppKey != "" {
		if device.akey, err = protocol.AESKeyFromString(device.AppKey); err != nil {
			http.Error(w, "AppKey incorrect format", http.StatusBadRequest)
			return
		}
		overrideAppKey = true
	}

	if !overrideAppKey {
		if device.akey, err = protocol.NewAESKey(); err != nil {
			logging.Warning("Unable to generate AppKey: %v", err)
			http.Error(w, "Unable to generate application key", http.StatusInternalServerError)
			return
		}
	}
	if device.AppSKey != "" {
		if device.askey, err = protocol.AESKeyFromString(device.AppSKey); err != nil {
			http.Error(w, "AppSKey incorrect format", http.StatusBadRequest)
			return
		}
		overrideAppSKey = true
	}
	if device.NwkSKey != "" {
		if device.nskey, err = protocol.AESKeyFromString(device.NwkSKey); err != nil {
			http.Error(w, "NwkSKey incorrect format", http.StatusBadRequest)
			return
		}
		overrideNwkSKey = true
	}
	if !overrideAppSKey {
		if device.askey, err = protocol.NewAESKey(); err != nil {
			logging.Warning("Unable to generate AppSKey: %v", err)
			http.Error(w, "Unable to generate application session key", http.StatusInternalServerError)
			return
		}
	}
	if !overrideNwkSKey {
		if device.nskey, err = protocol.NewAESKey(); err != nil {
			logging.Warning("Unable to generate NwkSKey: %v", err)
			http.Error(w, "Unable to generate network session key", http.StatusInternalServerError)
			return
		}
	}

	if deviceType == model.OverTheAirDevice && (overrideAppSKey || overrideDevAddr || overrideNwkSKey) {
		http.Error(w, "DevAddr, AppSKey and NwkSKey can only be specified for ABP devices", http.StatusBadRequest)
		return
	}
	if deviceType == model.PersonalizedDevice && overrideAppKey {
		http.Error(w, "AppKey can only be specified for OTAA devices", http.StatusBadRequest)
		return
	}

	// This might seem like a baroque way of getting an EUI but since EUIs can
	// be user-specified we will have EUIs that collide once in a while. Most of
	// the time this shouldn't be an issue but if large blocks are added there
	// might be more than one. This will attempt to create it for a relatively
	// small number of requests and skip the EUI counter forwards 10 steps at a
	// time.
	deviceToSave := device.ToModel(applicationEUI)

	attempts := 1
	devErr := storage.ErrAlreadyExists
	for devErr == storage.ErrAlreadyExists && attempts < 10 {
		devErr = s.context.Storage.Device.Put(deviceToSave, applicationEUI)
		if devErr == nil {
			break
		}
		if devErr == storage.ErrAlreadyExists && overrideEUI {
			http.Error(w, "Device EUI is already in use", http.StatusConflict)
			return
		}
		if devErr != storage.ErrAlreadyExists {
			logging.Warning("Unable to store device with EUI %s: %v", device.DeviceEUI, err)
			http.Error(w, "Unable to store device", http.StatusInternalServerError)
			return
		}
		logging.Warning("EUI (%s) for device is already in use. Trying another EUI.", deviceToSave.DeviceEUI)
		deviceToSave.DeviceEUI, err = s.context.KeyGenerator.NewDeviceEUI()
		if err != nil {
			http.Error(w, "Unable to create device identifier", http.StatusInternalServerError)
			return
		}
		attempts++
	}

	if devErr == storage.ErrAlreadyExists {
		logging.Error("Unable to find available EUI for device after 10 attempts")
		http.Error(w, "Unable to allocate device EUI", http.StatusInternalServerError)
		return
	}

	monitoring.DeviceCreated.Increment()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newDeviceFromModel(&deviceToSave)); err != nil {
		logging.Warning("Unable to marshal device with EUI %s: %v", deviceToSave.DeviceEUI, err)
	}
}

// deviceListHandler presents a list of devices associated with the application
func (s *Server) deviceListHandler(w http.ResponseWriter, r *http.Request) {
	applicationEUI, err := euiFromPathParameter(r, "aeui")
	if err != nil {
		http.Error(w, "Invalid application EUI", http.StatusBadRequest)
		return
	}

	// Retrieve application, make sure both network and application EUI is correct
	_, err = s.context.Storage.Application.GetByEUI(applicationEUI, s.connectUserID(r))
	if err != nil {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	switch r.Method {

	case http.MethodGet:
		s.deviceList(w, r, applicationEUI)

	case http.MethodPost:
		s.createDevice(w, r, applicationEUI)

	default:
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}
}

// Get and check EUIs for network, app, device. Returns false if one of the
// EUIs are malformed
func (s *Server) getDevice(w http.ResponseWriter, r *http.Request) (
	protocol.EUI, *model.Device) {

	app := s.getApplication(w, r)
	if app == nil {
		return protocol.EUI{}, nil
	}
	appEUI := app.AppEUI

	deviceEUI, err := euiFromPathParameter(r, "deui")
	if err != nil {
		http.Error(w, "Malformed Device EUI", http.StatusBadRequest)
		return appEUI, nil
	}

	device, err := s.context.Storage.Device.GetByEUI(deviceEUI)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return appEUI, nil
	}

	if device.AppEUI != appEUI {
		http.Error(w, "Application not found", http.StatusNotFound)
		return appEUI, nil
	}

	return appEUI, &device
}

// deviceInfoHandler shows info on a particular device
func (s *Server) deviceInfoHandler(w http.ResponseWriter, r *http.Request) {
	_, device := s.getDevice(w, r)
	if device == nil {
		return
	}

	switch r.Method {
	case http.MethodGet:

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newDeviceFromModel(device)); err != nil {
			logging.Warning("Unable to marshal device with EUI %s: %v", device.DeviceEUI, err)
		}

	case http.MethodPut:
		// Read request body as map of values
		var values map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
			logging.Info("Unable to decode JSON: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		var err error
		tmp, ok := values["devAddr"].(string)
		if ok {
			if device.DevAddr, err = protocol.DevAddrFromString(tmp); err != nil {
				http.Error(w, "Invalid DevAddr", http.StatusBadRequest)
				return
			}
		}
		oldApp, oldNet := device.AppSKey, device.NwkSKey
		tmp, ok = values["appKey"].(string)
		if ok {
			if device.AppKey, err = protocol.AESKeyFromString(tmp); err != nil {
				http.Error(w, "Invalid appKey", http.StatusBadRequest)
				return
			}
		}
		tmp, ok = values["appSKey"].(string)
		if ok {
			if device.AppSKey, err = protocol.AESKeyFromString(tmp); err != nil {
				http.Error(w, "Invalid appSKey", http.StatusBadRequest)
				return
			}
		}
		tmp, ok = values["nwkSKey"].(string)
		if ok {
			if device.NwkSKey, err = protocol.AESKeyFromString(tmp); err != nil {
				http.Error(w, "Invalid nwkSKey", http.StatusBadRequest)
				return
			}
		}
		// Even though just the NwkSKey have duplicates we'll have to change
		// both to reset the flag.
		if oldApp != device.AppSKey && oldNet != device.NwkSKey {
			device.KeyWarning = false
		}
		rc, ok := values["relaxedCounter"].(bool)
		if ok {
			device.RelaxedCounter = rc
		}
		fc, ok := values["fCntUp"].(int32)
		if ok {
			device.FCntUp = uint16(fc)
		}
		fc, ok = values["fCntDn"].(int32)
		if ok {
			device.FCntDn = uint16(fc)
		}
		kw, ok := values["keyWarning"].(bool)
		if ok {
			device.KeyWarning = kw
		}
		devType, ok := values["deviceType"].(string)
		if ok {
			switch strings.ToUpper(devType) {
			case "OTAA":
				_, ok := values["appKey"]
				if !ok {
					http.Error(w, "Must specify AppKey when changing device type to OTAA", http.StatusBadRequest)
				}
				device.State = model.OverTheAirDevice
			case "ABP":
				_, appS := values["appSKey"]
				_, nwkS := values["nwkSKey"]
				_, devA := values["devAddr"]
				if !appS || !nwkS || !devA {
					http.Error(w, "Must specify NwkSKey, AppSKey and DevAddr when changing device type to ABP", http.StatusBadRequest)
					return
				}
				device.State = model.PersonalizedDevice
			}
		}
		if !s.updateTags(&(device.Tags), values) {
			http.Error(w, "Invalid tag value", http.StatusBadRequest)
			return
		}
		if err := s.context.Storage.Device.Update(*device); err != nil {
			logging.Warning("Unable to update device with EUI %s: %v", device.DeviceEUI, err)
			http.Error(w, "Unable to update device", http.StatusInternalServerError)
			return
		}
		monitoring.DeviceUpdated.Increment()
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newDeviceFromModel(device)); err != nil {
			logging.Warning("Unable to marshal device with EUI %s to JSON: %v", device.DeviceEUI, err)
		}

	case http.MethodDelete:
		err := s.context.Storage.Device.Delete(device.DeviceEUI)
		switch err {
		case nil:
			monitoring.DeviceRemoved.Increment()
			w.WriteHeader(http.StatusNoContent)

		case storage.ErrNotFound:
			http.Error(w, "Device not found", http.StatusNotFound)

		default:
			http.Error(w, "Unable to remove device", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) deviceDataHandler(w http.ResponseWriter, r *http.Request) {
	// network EUI isn't checked except for a valid format.
	appEUI, device := s.getDevice(w, r)
	if device == nil {
		return
	}

	limit, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
	if err != nil {
		limit = int64(defaultMaxDeviceDataCount)
	}
	since, err := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	if err != nil {
		since = -1
	}
	switch r.Method {

	case http.MethodGet:
		ret, err := s.context.Storage.DeviceData.GetByDeviceEUI(device.DeviceEUI, int(limit))
		if err != nil {
			logging.Warning("Unable to retrieve data for device with EUI %s: %v", device.DeviceEUI, err)
			http.Error(w, "Unable to retrieve data for device", http.StatusInternalServerError)
			return
		}
		// Input in is ms, parameter is in ns
		if since > 0 {
			since = FromUnixMillis(since)
		}
		dataList := newAPIDataList()
		for data := range ret {
			if data.Timestamp < since {
				continue
			}
			dataList.Messages = append(dataList.Messages, newDeviceDataFromModel(data, appEUI))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(dataList); err != nil {
			logging.Warning("Unable to marshal device data for device with EUI %s: %v", device.DeviceEUI, err)
		}

	default:
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

// Remove downstream message if exists and completed (sent and/or acked).
// Returns false if there's an error. The existing message is left as is if
// it should be kept (ie it isn't sent yet or not acked yet). This might be
// a bit counter-intuitive but it will fail on the PutDownstream call later on
// this elminates a few obvious concurrency issues but not all.
func (s *Server) removeDownstreamIfComplete(w http.ResponseWriter, deviceEUI protocol.EUI) bool {
	existingMessage, err := s.context.Storage.DeviceData.GetDownstream(deviceEUI)
	if err == storage.ErrNotFound {
		return true
	}
	if err != nil {
		http.Error(w, "unable to verify if there's a scheduled message", http.StatusInternalServerError)
		return false
	}

	if !existingMessage.IsComplete() {
		http.Error(w, "a message is already scheduled for output", http.StatusConflict)
		return false
	}

	if err := s.context.Storage.DeviceData.DeleteDownstream(deviceEUI); err != nil {
		http.Error(w, "unable to remove scheduled message", http.StatusInternalServerError)
		return false
	}

	return true
}

func (s *Server) createDownstream(device *model.Device, w http.ResponseWriter, r *http.Request) {
	//
	// Read body, decode message
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logging.Warning("Unable to read request body for device %s: %v", device.DeviceEUI, err)
		http.Error(w, "Unable to read request body", http.StatusInternalServerError)
		return
	}
	outMessage := make(map[string]interface{})
	if err = json.Unmarshal(buf, &outMessage); err != nil {
		logging.Info("Unable to marshal JSON for message to device with EUI %s: %v", device.DeviceEUI, err)
		http.Error(w, "Can't grok JSON", http.StatusBadRequest)
		return
	}
	port, ok := outMessage["port"].(float64)
	if !ok {
		http.Error(w, "port must be set for messages", http.StatusBadRequest)
		return
	}
	if port < 1 || port > 223 {
		http.Error(w, "port must be between 1 and 223", http.StatusBadRequest)
		return
	}

	data, ok := outMessage["data"].(string)
	if !ok {
		http.Error(w, "data field must be set", http.StatusBadRequest)
	}
	payload, err := hex.DecodeString(data)
	if err != nil {
		http.Error(w, "Invalid data encoding. data should be encoded as a hex string", http.StatusBadRequest)
		return
	}
	if len(payload) == 0 {
		http.Error(w, "data cannot be zero bytes", http.StatusBadRequest)
		return
	}

	if !s.removeDownstreamIfComplete(w, device.DeviceEUI) {
		return
	}

	downstreamMsg := model.NewDownstreamMessage(device.DeviceEUI, uint8(port))
	downstreamMsg.Data = data

	ack, ok := outMessage["ack"].(bool)
	if ok {
		downstreamMsg.Ack = ack
	}
	if err := s.context.Storage.DeviceData.PutDownstream(device.DeviceEUI, downstreamMsg); err != nil {
		logging.Warning("Unable to store downstream message: %v", err)
		http.Error(w, "unable to schedule downstream message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newDownstreamMessageFromModel(downstreamMsg)); err != nil {
		logging.Warning("Unable to marshal downstream message for device with EUI %s into JSON: %v", device.DeviceEUI, err)
	}

}

func (s *Server) getDownstream(device *model.Device, w http.ResponseWriter, r *http.Request) {
	msg, err := s.context.Storage.DeviceData.GetDownstream(device.DeviceEUI)
	if err == storage.ErrNotFound {
		http.Error(w, "No downstream message scheduled for device", http.StatusNotFound)
		return
	}
	if err != nil {
		logging.Warning("Unable to retrieve downstream message: %v", err)
		http.Error(w, "Unable to retrieve downstream message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(newDownstreamMessageFromModel(msg)); err != nil {
		logging.Warning("Unable to marshal downstream message for device %s into JSON: %v", device.DeviceEUI, err)
	}
}

func (s *Server) deleteDownstream(device *model.Device, w http.ResponseWriter, r *http.Request) {
	if err := s.context.Storage.DeviceData.DeleteDownstream(device.DeviceEUI); err != nil && err != storage.ErrNotFound {
		logging.Warning("Unable to remove downstream message: %v", err)
		http.Error(w, "Unable to remove downstream message", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deviceSendHandler(w http.ResponseWriter, r *http.Request) {
	_, device := s.getDevice(w, r)
	if device == nil {
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.createDownstream(device, w, r)

	case http.MethodGet:
		s.getDownstream(device, w, r)

	case http.MethodDelete:
		s.deleteDownstream(device, w, r)

	default:
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
	return
}

func euiToSource(eui protocol.EUI) string {
	return fmt.Sprintf("0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x",
		eui.Octets[0], eui.Octets[1], eui.Octets[2], eui.Octets[3],
		eui.Octets[4], eui.Octets[5], eui.Octets[6], eui.Octets[7])
}

func keyToSource(key protocol.AESKey) string {
	return fmt.Sprintf("0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x",
		key.Key[0], key.Key[1], key.Key[2], key.Key[3],
		key.Key[4], key.Key[5], key.Key[6], key.Key[7],
		key.Key[8], key.Key[9], key.Key[10], key.Key[11],
		key.Key[12], key.Key[13], key.Key[14], key.Key[15])
}

func (s *Server) deviceSourceHandler(w http.ResponseWriter, r *http.Request) {
	_, device := s.getDevice(w, r)
	if device == nil {
		return
	}

	params := templateParameters{}
	if device.State == model.PersonalizedDevice {
		params.DeviceEUI = euiToSource(protocol.EUI{})
		params.AppKey = keyToSource(protocol.AESKey{})
		params.AppEUI = euiToSource(protocol.EUI{})
		params.DevAddr = fmt.Sprintf("0x%s", device.DevAddr)
		params.NwkSKey = keyToSource(device.NwkSKey)
		params.AppSKey = keyToSource(device.AppSKey)
		params.OTAA = false
	} else {
		application, err := s.context.Storage.Application.GetByEUI(device.AppEUI, s.connectUserID(r))
		if err != nil {
			logging.Warning("Unable to retrieve application with EUI %s: %v", device.AppEUI, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		params.DeviceEUI = euiToSource(device.DeviceEUI)
		params.AppKey = keyToSource(device.AppKey)
		params.AppEUI = euiToSource(application.AppEUI)
		params.DevAddr = "0x0"
		params.AppSKey = keyToSource(protocol.AESKey{})
		params.NwkSKey = keyToSource(protocol.AESKey{})
		params.OTAA = true
	}

	templateType := r.URL.Query().Get("type")
	templateText := getSourceTemplate(templateType)
	t := template.Must(template.New("source_" + templateType).Parse(templateText))
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, params); err != nil {
		logging.Warning("Unable to execute template: %v", err)
	}
}
