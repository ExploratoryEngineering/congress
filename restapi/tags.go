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
	"net/url"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
	"github.com/telenordigital/goconnect"
)

// tagListResource lists tag on a resource. This method is common for all
// entities with tags.
func (h *Server) tagListResource(w http.ResponseWriter, r *http.Request, tags model.Tags) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(tags.TagJSON())
}

// tagPostResource handles POST requests to the tag resource, ie create a new
// tag. If the function returns true the tag has been added successfully (and the
// tags should be stored in the storage backend)
func (h *Server) tagPostResource(w http.ResponseWriter, r *http.Request, tags model.Tags) bool {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logging.Warning("Unable to read body for tags: %v", err)
		http.Error(w, "Unable to read body for request", http.StatusInternalServerError)
		return false
	}
	values := make(map[string]string)
	if err := json.Unmarshal(buf, &values); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return false
	}
	for k, v := range values {
		if tags.Exists(k) {
			http.Error(w, "Tag already exists", http.StatusConflict)
			return false
		}
		if err := tags.SetTag(k, v); err == model.ErrInvalidChars {
			http.Error(w, "Invalid characters in name and/or value", http.StatusBadRequest)
			return false
		}
	}
	return true
}

// tagNameResource handles GET and DELETE requests to a particular tag. If
// it returns true the tag has been changed (and the tags should be stored in
// the storage backend)
func (h *Server) tagNameResource(w http.ResponseWriter, r *http.Request, tags model.Tags) bool {
	unescapedName, ok := r.Context().Value(pathParameter("name")).(string)
	if !ok {
		http.Error(w, "Missing name from path", http.StatusInternalServerError)
		return false
	}
	name, err := url.QueryUnescape(unescapedName)
	if err != nil {
		http.Error(w, "Invalid tag name", http.StatusBadRequest)
		return false
	}
	val, ok := tags.GetTag(name)
	if !ok {
		http.Error(w, "Tag not found", http.StatusNotFound)
		return false
	}

	switch r.Method {

	case http.MethodGet:
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(val))

	case http.MethodDelete:
		tags.RemoveTag(name)
		return true
	}
	return false
}

// Handle GET and POST requests to the tag resource. If true is returned the
// tags are mutated.
func (h *Server) tagCollectionHandler(w http.ResponseWriter, r *http.Request, tags model.Tags) bool {
	switch r.Method {

	case http.MethodGet:
		h.tagListResource(w, r, tags)
		return false

	case http.MethodPost:
		return h.tagPostResource(w, r, tags)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
}

// Helper function to get the gateway EUI via the request path. Returns nil if
// the gateway can't be found. Error messages are sent to thte client.
func (h *Server) getGateway(w http.ResponseWriter, r *http.Request) *model.Gateway {
	eui, err := euiFromPathParameter(r, "geui")
	if err != nil {
		http.Error(w, "Invalid EUI", http.StatusBadRequest)
		return nil
	}
	gw, err := h.context.Storage.Gateway.Get(eui, h.connectUserID(r))
	if err != nil {
		if err == storage.ErrNotFound {
			http.Error(w, "Gateway not found", http.StatusNotFound)
			return nil
		}
		http.Error(w, "Unable to load gateway", http.StatusInternalServerError)
		return nil
	}
	return &gw
}

// ----------------------------------------------------------------------------
// Gateway

// Handler for the gateway's tags resource
func (h *Server) gatewayTagHandler(w http.ResponseWriter, r *http.Request) {
	gw := h.getGateway(w, r)
	if gw == nil {
		return
	}
	if h.tagCollectionHandler(w, r, gw.Tags) {
		if err := h.context.Storage.Gateway.Update(*gw, h.connectUserID(r)); err != nil {
			http.Error(w, "Unable to store tags", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(gw.Tags.TagJSON())
	}
}

// Handler for the gateways tags/<name> resources
func (h *Server) gatewayTagNameHandler(w http.ResponseWriter, r *http.Request) {
	gw := h.getGateway(w, r)
	if gw == nil {
		return
	}

	if h.tagNameResource(w, r, gw.Tags) {
		if err := h.context.Storage.Gateway.Update(*gw, h.connectUserID(r)); err != nil {
			http.Error(w, "Unable to update tag", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ----------------------------------------------------------------------------
// Application

// Handler for the application tags resource
func (h *Server) applicationTagHandler(w http.ResponseWriter, r *http.Request) {
	app := h.getApplication(w, r)
	if app == nil {
		return
	}

	if h.tagCollectionHandler(w, r, app.Tags) {
		if err := h.context.Storage.Application.Update(*app, h.connectUserID(r)); err != nil {
			http.Error(w, "Unable to update tags", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(app.Tags.TagJSON())
	}
}

func (h *Server) applicationTagNameHandler(w http.ResponseWriter, r *http.Request) {
	app := h.getApplication(w, r)
	if app == nil {
		return
	}

	if h.tagNameResource(w, r, app.Tags) {
		if err := h.context.Storage.Application.Update(*app, h.connectUserID(r)); err != nil {
			http.Error(w, "Unable to remove tag", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ----------------------------------------------------------------------------
// Device
func (h *Server) deviceTagHandler(w http.ResponseWriter, r *http.Request) {
	_, device := h.getDevice(w, r)
	if device == nil {
		return
	}

	if h.tagCollectionHandler(w, r, device.Tags) {
		if err := h.context.Storage.Device.Update(*device); err != nil {
			http.Error(w, "Unable to store tags", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(device.Tags.TagJSON())
	}
}

func (h *Server) deviceTagNameHandler(w http.ResponseWriter, r *http.Request) {
	_, device := h.getDevice(w, r)
	if device == nil {
		return
	}

	if h.tagNameResource(w, r, device.Tags) {
		if err := h.context.Storage.Device.Update(*device); err != nil {
			http.Error(w, "Unable to update device", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *Server) ensureSessionAndToken(w http.ResponseWriter, r *http.Request) (*model.APIToken, *goconnect.Session, bool) {
	s := r.Context().Value(goconnect.SessionContext)
	session, ok := s.(goconnect.Session)
	if !ok {
		http.Error(w, "No session found", http.StatusUnauthorized)
		return nil, nil, false
	}
	t := r.Context().Value(pathParameter("token"))
	tokenStr, ok := t.(string)
	if !ok {
		// This shouldn't happen but it is a nice failsafe
		logging.Warning("Missing token from request path. Is the route configured properly?")
		http.Error(w, "Missing token in path", http.StatusBadRequest)
		return nil, nil, false
	}
	token, err := h.context.Storage.Token.Get(tokenStr)
	if err != nil {
		http.Error(w, "Token not found", http.StatusNotFound)
		return nil, nil, false
	}
	return &token, &session, true
}

func (h *Server) tokenTagHandler(w http.ResponseWriter, r *http.Request) {
	token, session, success := h.ensureSessionAndToken(w, r)
	if !success {
		return
	}
	if h.tagCollectionHandler(w, r, token.Tags) {
		if err := h.context.Storage.Token.Update(*token, model.UserID(session.UserID)); err != nil {
			http.Error(w, "Unable to store tags", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(token.Tags.TagJSON())
	}
}

func (h *Server) tokenTagNameHandler(w http.ResponseWriter, r *http.Request) {
	token, session, success := h.ensureSessionAndToken(w, r)
	if !success {
		return
	}
	if h.tagNameResource(w, r, token.Tags) {
		if err := h.context.Storage.Token.Update(*token, model.UserID(session.UserID)); err != nil {
			http.Error(w, "Unable to update device", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
