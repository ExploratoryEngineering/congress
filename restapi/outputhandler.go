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
	"net/http"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

// These are the app output resources for applications.

func (h *Server) getOutput(w http.ResponseWriter, r *http.Request, appEUI protocol.EUI) *model.AppOutput {
	outputEUI, err := euiFromPathParameter(r, "oeui")
	if err != nil {
		http.Error(w, "Malformed Output EUI", http.StatusBadRequest)
		return nil
	}

	ch, err := h.context.Storage.AppOutput.GetByApplication(appEUI)
	if err != nil {
		if err != storage.ErrNotFound {
			logging.Warning("Unable to retrieve output for application %s: %v", appEUI, err)
		}
		http.Error(w, "Output not found", http.StatusNotFound)
		return nil
	}

	var ret model.AppOutput
	found := false
	for v := range ch {
		if v.EUI == outputEUI {
			ret = v
			found = true
		}
	}
	if !found {
		http.Error(w, "Output not found", http.StatusNotFound)
		return nil
	}

	return &ret
}

func (h *Server) outputHandler(w http.ResponseWriter, r *http.Request) {
	app := h.getApplication(w, r)
	if app == nil {
		return
	}

	switch r.Method {
	case http.MethodGet:
		var ret apiAppOutputList
		ch, err := h.context.Storage.AppOutput.GetByApplication(app.AppEUI)
		if err != nil && err != storage.ErrNotFound {
			logging.Warning("Unable to retrive list of outputs for app with EUI %s: %v", app.AppEUI, err)
			http.Error(w, "Unable to retrieve list of ouputs", http.StatusInternalServerError)
			return
		}
		ret.List = make([]apiAppOutput, 0)
		for op := range ch {
			status, logs, err := h.context.AppOutput.GetStatusAndLogs(&op)
			if err != nil {
				logging.Warning("Unable to get status for output with eui %s", op.EUI)
				status = "indeterminate"
				tmplog := server.NewMemoryLogger()
				tmplog.Append(server.NewLogEntry("Output isn't running"))
				logs = &tmplog
			}
			ret.List = append(ret.List, newOutputFromModel(op, logs, status))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ret); err != nil {
			logging.Warning("Unable to marshal output list into JSON: %v", err)
		}

	case http.MethodPost:
		newItem := apiAppOutput{}
		if err := json.NewDecoder(r.Body).Decode(&newItem); err != nil {
			http.Error(w, "Invalid JSON in request", http.StatusBadRequest)
			return
		}
		newItem.AppEUI = app.AppEUI.String()
		newItemEUI, err := h.context.KeyGenerator.NewOutputEUI()
		if err != nil {
			http.Error(w, "Key space exhausted", http.StatusInternalServerError)
			return
		}
		newItem.EUI = newItemEUI.String()
		newAppOutput, err := newItem.ToModel()
		if err != nil {
			http.Error(w, "Invalid configuration", http.StatusBadRequest)
			return
		}
		if err := h.context.AppOutput.Add(&newAppOutput); err != nil {
			if err == server.ErrInvalidTransport {
				http.Error(w, "Invalid output configuration", http.StatusBadRequest)
				return
			}
			logging.Warning("Unable to add new output for app with EUI %s: %v", app.AppEUI, err)
			http.Error(w, "Unable to start output", http.StatusInternalServerError)
			return
		}
		if err := h.context.Storage.AppOutput.Put(newAppOutput); err != nil {
			http.Error(w, "Unable to store output", http.StatusInternalServerError)
			// Remove the launched output
			h.context.AppOutput.Remove(&newAppOutput)
			return
		}
		status, logger, err := h.context.AppOutput.GetStatusAndLogs(&newAppOutput)
		if err != nil {
			logging.Warning("Unable to retrieve log for output with EUI %s: %v", newAppOutput.EUI, err)
			http.Error(w, "Unable to retrieve output status", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(newOutputFromModel(newAppOutput, logger, status)); err != nil {
			logging.Warning("Unable to marshal output with EUI %s into JSON: %v", newAppOutput.EUI, err)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

	}
}

// Show the state and configuration of a single output
func (h *Server) outputInfoHandler(w http.ResponseWriter, r *http.Request) {
	app := h.getApplication(w, r)
	if app == nil {
		return
	}

	op := h.getOutput(w, r, app.AppEUI)
	if op == nil {
		return
	}

	switch r.Method {

	case http.MethodGet:
		status, logger, err := h.context.AppOutput.GetStatusAndLogs(op)
		if err != nil {
			logging.Warning("Unable to retrieve status and logs for output with eui %s: %v", op.EUI, err)
			http.Error(w, "Unable to retrieve output status", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newOutputFromModel(*op, logger, status)); err != nil {
			logging.Warning("Unable to marshal output with EUI %s into JSON: %v", op.EUI, err)
		}

	case http.MethodPut:
		// Retrieve the configuration and update it.
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logging.Info("Unable to read request body for output with EUI %s: %v", op.EUI, err)
			http.Error(w, "Unable to read request body", http.StatusInternalServerError)
			return
		}
		updatedOutput := apiAppOutput{}
		if err := json.Unmarshal(buf, &updatedOutput); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		op.Configuration = updatedOutput.Config
		if err := h.context.Storage.AppOutput.Update(*op); err != nil {
			logging.Warning("Unable to store output with EUI: %v", op.EUI, err)
			http.Error(w, "Unable to store output", http.StatusInternalServerError)
			return
		}
		if err := h.context.AppOutput.Update(op); err != nil {
			logging.Warning("Unable to update output with EUI %s: %v", op.EUI, err)
			http.Error(w, "Unable to update output", http.StatusInternalServerError)
			return
		}
		status, logs, err := h.context.AppOutput.GetStatusAndLogs(op)
		if err != nil {
			logging.Warning("Unable to retrieve output status for output with eui %s: %v", op.EUI, err)
			http.Error(w, "Can't determine status of output", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newOutputFromModel(*op, logs, status)); err != nil {
			logging.Warning("Unable to marshal output with EUI %s into JSON: %v", op.EUI, err)
		}

	case http.MethodDelete:
		if err := h.context.Storage.AppOutput.Delete(*op); err != nil {
			logging.Warning("Unable to remove output: %v", err)
			http.Error(w, "Can't remove output", http.StatusInternalServerError)
			return
		}
		// it's OK if it isn't running
		if err := h.context.AppOutput.Remove(op); err != nil && err != server.ErrNotFound {
			logging.Warning("Unable to stop output: %v", err)
			http.Error(w, "Can't stop output", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

	}
}
