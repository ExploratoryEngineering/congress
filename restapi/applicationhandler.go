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
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/ExploratoryEngineering/congress/monitoring"

	"golang.org/x/net/websocket"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

// The maximum number of data packets to return from the .../data endpoint
const defaultMaxDeviceDataCount int = 50

// Read application from request body. Emits error message to client if there's an error
func (s *Server) readAppFromRequest(w http.ResponseWriter, r *http.Request) (apiApplication, error) {
	buf, err := ioutil.ReadAll(r.Body)
	app := apiApplication{}

	if err != nil {
		logging.Warning("Unable to read request body from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Unable to read request body", http.StatusInternalServerError)
		return app, err
	}

	if err := json.Unmarshal(buf, &app); err != nil {
		logging.Info("Unable to unmarshal JSON: %v, (%s)", err, string(buf))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return app, err
	}
	return app, nil
}

// Handle POST to application collection, ie create application
func (s *Server) createApplication(w http.ResponseWriter, r *http.Request) {
	application, err := s.readAppFromRequest(w, r)
	if err != nil {
		return
	}

	var overrideEUI bool
	if application.ApplicationEUI != "" {
		overrideEUI = true
		if application.eui, err = protocol.EUIFromString(application.ApplicationEUI); err != nil {
			http.Error(w, "Invalid EUI format", http.StatusBadRequest)
			return
		}
	}
	if !overrideEUI {
		if application.eui, err = s.context.KeyGenerator.NewAppEUI(); err != nil {
			logging.Warning("Unable to generate application EUI: %v", err)
			http.Error(w, "Unable to create application EUI", http.StatusInternalServerError)
			return
		}
	}
	application.ApplicationEUI = application.eui.String()

	// This might seem like a baroque way of getting an EUI but since EUIs can
	// be user-specified we will have EUIs that collide once in a while. Most of
	// the time this shouldn't be an issue but if large blocks are added there
	// might be more than one. This will attempt to create it for a relatively
	// small number of requests and skip the EUI counter forwards 10 steps at a
	// time.
	appErr := storage.ErrAlreadyExists
	app := application.ToModel()
	userID := s.connectUserID(r)

	attempts := 1
	for appErr == storage.ErrAlreadyExists && attempts < 10 {
		appErr = s.context.Storage.Application.Put(app, userID)
		if appErr == nil {
			break
		}
		if appErr == storage.ErrAlreadyExists && overrideEUI {
			// Can't reuse app EUI
			http.Error(w, "Application EUI is already in use", http.StatusConflict)
			return
		}
		if appErr != storage.ErrAlreadyExists {
			// Some other error - fail with 500
			logging.Warning("Unable to store application: %s", err)
			http.Error(w, "Unable to store application", http.StatusInternalServerError)
			return
		}

		logging.Warning("EUI (%s) for application is already in use. Trying another EUI.", app.AppEUI)
		app.AppEUI, err = s.context.KeyGenerator.NewAppEUI()
		if err != nil {
			http.Error(w, "Unable to generate application identifier", http.StatusInternalServerError)
			return
		}
		attempts++
	}
	if appErr == storage.ErrAlreadyExists {
		logging.Error("Unable to create find available EUI even after 10 attempts. Returning error to client")
		http.Error(w, "Unable to store application", http.StatusInternalServerError)
		return
	}

	// Set the tags property if it isn't already set
	if application.Tags == nil {
		application.Tags = make(map[string]string)
	}

	monitoring.ApplicationCreated.Increment()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(application); err != nil {
		logging.Warning("Unable to marshal application object: %v", err)
	}
}

// Handle GET on application collection, ie list applications
func (s *Server) applicationList(w http.ResponseWriter, r *http.Request) {
	// GET returns a JSON array with applications.
	applications, err := s.context.Storage.Application.GetList(s.connectUserID(r))
	if err != nil {
		logging.Warning("Unable to read application list: %v", err)
		http.Error(w, "Unable to load applications", http.StatusInternalServerError)
		return
	}
	appList := newApplicationList()
	for application := range applications {
		appList.Applications = append(appList.Applications, newAppFromModel(application))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(appList); err != nil {
		logging.Warning("Unable to marshal application object: %v", err)
	}
}

// Shows a list of applications.
func (s *Server) applicationListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		s.applicationList(w, r)

	case http.MethodPost:
		s.createApplication(w, r)

	default:
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) getApplication(w http.ResponseWriter, r *http.Request) *model.Application {
	appEUI, err := euiFromPathParameter(r, "aeui")
	if err != nil {
		http.Error(w, "Malformed Application EUI", http.StatusBadRequest)
		return nil
	}
	application, err := s.context.Storage.Application.GetByEUI(appEUI, s.connectUserID(r))
	if err != nil {
		http.Error(w, "Application not found", http.StatusNotFound)
		return nil
	}
	return &application
}

func (s *Server) removeAppOutputs(appEUI protocol.EUI) error {
	outputs, err := s.context.Storage.AppOutput.GetByApplication(appEUI)
	if err == storage.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	for output := range outputs {
		if err := s.context.AppOutput.Remove(&output); err != nil {
			logging.Warning("Unable to remote app output %s for application %s: %v", output.EUI, appEUI, err)
		}
	}
	return nil
}

// Return a single application formatted as JSON.
func (s *Server) applicationInfoHandler(w http.ResponseWriter, r *http.Request) {
	application := s.getApplication(w, r)
	if application == nil {
		return
	}

	switch r.Method {

	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newAppFromModel(*application)); err != nil {
			logging.Warning("Unable to marshal application with EUI %s into JSON: %v", application.AppEUI, err)
		}

	case http.MethodPut:
		var err error
		var values map[string]interface{}
		if err = json.NewDecoder(r.Body).Decode(&values); err != nil {
			logging.Info("Unable to unmarshal request from %s: %v", r.RemoteAddr, err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if !s.updateTags(&application.Tags, values) {
			http.Error(w, "Invalid tag value", http.StatusBadRequest)
			return
		}

		if err := s.context.Storage.Application.Update(*application, s.connectUserID(r)); err != nil {
			// We already know if the application doesn't exist at this point so updates
			// should succeed (ignoring ignoring ErrNotFound return)
			logging.Warning("Unable to update application: %v", err)
			http.Error(w, "Unable to update application", http.StatusInternalServerError)
			return
		}
		// Success - return the modified application
		monitoring.ApplicationUpdated.Increment()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newAppFromModel(*application)); err != nil {
			logging.Warning("Unable to marshal application with EUI %s into JSON: %v", application.AppEUI, err)
		}
		return

	case http.MethodDelete:
		if err := s.removeAppOutputs(application.AppEUI); err != nil {
			http.Error(w, "Unable to remove outputs", http.StatusInternalServerError)
			return
		}
		err := s.context.Storage.Application.Delete(application.AppEUI, s.connectUserID(r))
		switch err {
		case nil:
			monitoring.ApplicationRemoved.Increment()
			monitoring.RemoveAppCounters(application.AppEUI)
			w.WriteHeader(http.StatusNoContent)
		case storage.ErrNotFound:
			// This is covered above but race conditions might apply here
			http.Error(w, "Application not found", http.StatusNotFound)
		case storage.ErrDeleteConstraint:
			http.Error(w, "Application can't be deleted", http.StatusConflict)
		default:
			logging.Warning("Unable to delete application: %v", err)
			http.Error(w, "Unable to delete application", http.StatusInternalServerError)
		}
		return

	default:
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
	}
}

func (s *Server) applicationDataHandler(w http.ResponseWriter, r *http.Request) {
	application := s.getApplication(w, r)
	if application == nil {
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
		// Since parameter is in ms, time stamp is in ns
		if since > 0 {
			since = FromUnixMillis(since)
		}
		ret, err := s.context.Storage.DeviceData.GetByApplicationEUI(application.AppEUI, int(limit))
		if err != nil {
			logging.Warning("Unable to retrieve data for application %s: %v", application.AppEUI, err)
			http.Error(w, "Unable to retrieve data for application", http.StatusInternalServerError)
			return
		}

		dataList := newAPIDataList()
		for data := range ret {
			if data.Timestamp < since {
				continue
			}
			dataList.Messages = append(dataList.Messages, newDeviceDataFromModel(data, application.AppEUI))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(dataList); err != nil {
			logging.Warning("Unable to marshal application data (EUI: %s): %v", application.AppEUI, err)
		}

	default:
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

// Websocket error function. Writes error to the websocket and closes it.
func writeError(ws *websocket.Conn, msg string) {
	if err := json.NewEncoder(ws).Encode(newWSError(msg)); err != nil {
		logging.Warning("Unable to write and marshal WS error JSON: %v", err)
	}
}

// Websocket handler. Subscribes to messages for the given EUI and forwards them
// to the websocket.
func (s *Server) applicationWebsocketHandler(ws *websocket.Conn) {
	defer ws.Close()
	appEUI, err := euiFromPathParameter(ws.Request(), "aeui")
	if nil != err {
		writeError(ws, err.Error())
		return
	}
	_, err = s.context.Storage.Application.GetByEUI(appEUI, s.connectUserID(ws.Request()))
	if err == storage.ErrNotFound {
		writeError(ws, "Unknown application EUI")
		return
	}
	if err != nil {
		writeError(ws, "Unable to read application")
		logging.Warning("Unable to read application with EUI %s: %v", appEUI, err)
		return
	}

	ch := s.context.AppRouter.Subscribe(appEUI)
	defer s.context.AppRouter.Unsubscribe(ch)

	for {
		select {
		case p := <-ch:
			message, ok := p.(*server.PayloadMessage)
			if !ok {
				logging.Error("Expected type %T on channel but got the type %T. Publisher error?", message, p)
				continue
			}
			deviceMessage := newWSData(&apiDeviceData{
				DevAddr:    message.Device.DevAddr.String(),
				Timestamp:  ToUnixMillis(message.FrameContext.GatewayContext.ReceivedAt.UnixNano()),
				Data:       hex.EncodeToString(message.Payload),
				AppEUI:     message.Application.AppEUI.String(),
				DeviceEUI:  message.Device.DeviceEUI.String(),
				RSSI:       message.FrameContext.GatewayContext.Radio.RSSI,
				SNR:        message.FrameContext.GatewayContext.Radio.SNR,
				Frequency:  message.FrameContext.GatewayContext.Radio.Frequency,
				DataRate:   message.FrameContext.GatewayContext.Radio.DataRate,
				GatewayEUI: message.FrameContext.GatewayContext.Gateway.GatewayEUI.String(),
			},
			)

			if err := json.NewEncoder(ws).Encode(deviceMessage); err != nil {
				return
			}

		case <-time.After(30 * time.Second):
			if err := json.NewEncoder(ws).Encode(newWSKeepAlive()); err != nil {
				logging.Info("Unable to send keepalive to websocket at %v. Closing web socket", ws.RemoteAddr())
				return
			}
		}
	}
}

func (s *Server) applicationStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	eui, err := euiFromPathParameter(r, "aeui")
	if err != nil {
		http.Error(w, "Invalid EUI", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(monitoring.GetAppCounters(eui))
}
