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
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ExploratoryEngineering/congress/events/gwevents"
	"github.com/ExploratoryEngineering/congress/monitoring"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
	"golang.org/x/net/websocket"
)

func (s *Server) gatewayList(w http.ResponseWriter, r *http.Request) {
	gateways, err := s.context.Storage.Gateway.GetList(s.connectUserID(r))
	if err != nil {
		logging.Warning("Unable to get list of gateways: %v", err)
		http.Error(w, "Unable to read list of gateways", http.StatusInternalServerError)
		return
	}

	gatewayList := newGatewayList()
	for gateway := range gateways {
		gatewayList.Gateways = append(gatewayList.Gateways, newGatewayFromModel(gateway))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(gatewayList); err != nil {
		logging.Warning("Unable to marshal gateway list: %v", err)
	}
}

func (s *Server) createGateway(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logging.Info("Unable to read request body: %v", err)
		http.Error(w, "Unable to read request", http.StatusInternalServerError)
		return
	}

	gateway := apiGateway{}
	if err := json.Unmarshal(buf, &gateway); err != nil {
		logging.Info("Unable to unmarshal JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(gateway.GatewayEUI) == "" {
		http.Error(w, "Missing gateway EUI", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(gateway.IP) == "" {
		http.Error(w, "Missing gateway IP", http.StatusBadRequest)
		return
	}

	if gateway.eui, err = protocol.EUIFromString(gateway.GatewayEUI); err != nil {
		http.Error(w, "Invalid gateway EUI", http.StatusBadRequest)
		return
	}
	if gateway.ipaddr = net.ParseIP(gateway.IP); gateway.ipaddr == nil {
		http.Error(w, "Invalid IP format", http.StatusBadRequest)
		return
	}

	// Sanity check the lat/lon coordinates
	if gateway.Longitude > 360 || gateway.Longitude < -360 ||
		gateway.Latitude < -90 || gateway.Latitude > 90 {
		http.Error(w, "Longitude must be [180.0, 180] Latitude must be and [-90, 90]",
			http.StatusBadRequest)
		return
	}

	modelGw := gateway.ToModel()
	if err = s.context.Storage.Gateway.Put(gateway.ToModel(), s.connectUserID(r)); err != nil {
		if err == storage.ErrAlreadyExists {
			http.Error(w, "A gateway with that EUI alread exists", http.StatusConflict)
			return
		}
		logging.Warning("Unable to store gateway with EUI %s: %v", gateway.GatewayEUI, err)
		http.Error(w, "Unable to store gateway", http.StatusInternalServerError)
		return
	}

	monitoring.GatewayCreated.Increment()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newGatewayFromModel(modelGw)); err != nil {
		logging.Warning("Unable to marshal gateway with EUI %s into JSON: %v", modelGw.GatewayEUI, err)
	}
}

func (s *Server) gatewayListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.gatewayList(w, r)

	case http.MethodPost:
		s.createGateway(w, r)

	default:
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
	}
}

func (s *Server) gatewayInfoHandler(w http.ResponseWriter, r *http.Request) {
	eui, err := euiFromPathParameter(r, "geui")
	if err != nil {
		logging.Debug("Unable to convert EUI from string: %v", err)
		http.Error(w, "Invalid EUI", http.StatusBadRequest)
		return
	}

	modelGateway, err := s.context.Storage.Gateway.Get(eui, s.connectUserID(r))
	if err != nil {
		logging.Info("Unable to look up gateway with EUI %s: %v", eui, err)
		http.Error(w, "Gateway not found", http.StatusNotFound)
		return
	}

	switch r.Method {

	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newGatewayFromModel(modelGateway)); err != nil {
			logging.Warning("Unable to marshal gateway with EUI %s into JSON: %v", modelGateway.GatewayEUI, err)
		}
		return

	case http.MethodPut:
		var values map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		alt, ok := values["altitude"].(float64)
		if ok {
			modelGateway.Altitude = float32(alt)
		}
		lat, ok := values["latitude"].(float64)
		if ok {
			modelGateway.Latitude = float32(lat)
		}
		lon, ok := values["longitude"].(float64)
		if ok {
			modelGateway.Longitude = float32(lon)
		}
		ip, ok := values["ip"].(string)
		if ok {
			if modelGateway.IP = net.ParseIP(ip); modelGateway.IP == nil {
				http.Error(w, "Invalid IP address", http.StatusBadRequest)
				return
			}
		}
		strict, ok := values["strictIP"].(bool)
		if ok {
			modelGateway.StrictIP = strict
		}

		if !s.updateTags(&(modelGateway.Tags), values) {
			http.Error(w, "Invalid tag name or value", http.StatusBadRequest)
			return
		}
		if err := s.context.Storage.Gateway.Update(modelGateway, s.connectUserID(r)); err != nil {
			logging.Warning("Unable to update gateway: %v", err)
			http.Error(w, "Unable to update gateway", http.StatusInternalServerError)
			return
		}
		monitoring.GatewayUpdated.Increment()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newGatewayFromModel(modelGateway)); err != nil {
			logging.Warning("Unable to marshal gateway with EUI %s into JSON: %v", modelGateway.GatewayEUI, err)
		}

	case http.MethodDelete:
		if err := s.context.Storage.Gateway.Delete(eui, s.connectUserID(r)); err != nil {
			logging.Warning("Unable to delete gateway: %v", err)
			http.Error(w, "Unable to remove gateway", http.StatusInternalServerError)
			return
		}
		monitoring.GatewayRemoved.Increment()
		monitoring.RemoveGatewayCounters(eui)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handler for websocket with live view from gateway
func (s *Server) gatewayWebsocketHandler(ws *websocket.Conn) {
	defer ws.Close()
	eui, err := euiFromPathParameter(ws.Request(), "geui")
	if err != nil {
		logging.Info("Unable to retrieve EUI parameter from path: %v", err)
		writeError(ws, "Invalid gateway EUI")
		return
	}
	_, err = s.context.Storage.Gateway.Get(eui, s.connectUserID(ws.Request()))
	if err != nil {
		if err != storage.ErrNotFound {
			logging.Warning("Unable to read gateway with EUI %s: %v", eui, err)
		}
		writeError(ws, "Gateway not found")
		return
	}

	gwEventChannel := s.context.GwEventRouter.Subscribe(eui)

	defer s.context.GwEventRouter.Unsubscribe(gwEventChannel)

	for {
		var result interface{}
		select {
		case result = <-gwEventChannel:
		case <-time.After(60 * time.Second):
			// No event for 60 seconds -- send inactive event
			result = gwevents.NewInactive()
		}

		if err := json.NewEncoder(ws).Encode(result); err != nil {
			logging.Warning("Unable to marshal JSON for websocket: %v", err)
			return
		}
	}
}

func (s *Server) gatewayStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	eui, err := euiFromPathParameter(r, "geui")
	if err != nil {
		http.Error(w, "Invalid EUI", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(monitoring.GetGatewayCounters(eui))
}

func (s *Server) gatewayPublicList(w http.ResponseWriter, r *http.Request) {
	gateways, err := s.context.Storage.Gateway.ListAll()
	if err != nil {
		logging.Warning("Unable to get list of gateways: %v", err)
		http.Error(w, "Unable to read list of gateways", http.StatusInternalServerError)
		return
	}

	var list apiPublicGatewayList
	for gateway := range gateways {
		list.Gateways = append(list.Gateways, newPublicGatewayFromModel(gateway))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(list); err != nil {
		logging.Warning("Unable to marshal gateway list: %v", err)
	}
}
