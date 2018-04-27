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
	"strings"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
	"github.com/telenordigital/goconnect"
)

const tokenHeaderName = "X-API-Token"

// isValidToken checks if the API token is valid and applies to the method and
// path. It will return a triplet with success (true/false), error message and
// HTTP status code. This *could* have been a filter for the request but the
// resources use the ResponseWriter type and that is a can of worms wrt the
//
func (h *Server) isValidToken(headerToken, method string, path string) (bool, string, int, model.UserID) {
	if headerToken == "" {
		return false, "Missing API Token", http.StatusUnauthorized, model.InvalidUserID
	}
	// Ensure the token exists.
	token, err := h.context.Storage.Token.Get(headerToken)
	if err == storage.ErrNotFound {
		return false, "Invalid API token", http.StatusUnauthorized, model.InvalidUserID
	}
	if err != nil {
		logging.Warning("Unable to retrieve token: %v", err)
		return false, "Token error", http.StatusInternalServerError, model.InvalidUserID
	}

	// Ensure the token matches the path
	if !strings.HasPrefix(path, token.Resource) {
		return true, "Invalid path", http.StatusForbidden, model.InvalidUserID
	}

	// Block POST, DELETE and GET unless it is a write operation
	if !token.Write && (method == http.MethodPatch ||
		method == http.MethodPost || method == http.MethodDelete) {
		return true, "Access error", http.StatusForbidden, model.InvalidUserID
	}

	// Everything checks out OK
	return true, "", http.StatusOK, token.UserID
}

// tokenInfoHandler handles GET and DELETE requests to a particular token
func (h *Server) tokenInfoHandler(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value(pathParameter("token"))
	token, ok := t.(string)
	if !ok {
		// This shouldn't happen but it is a nice failsafe
		logging.Warning("Missing token from request path. Is the route configured properly?")
		http.Error(w, "Missing token in path", http.StatusBadRequest)
		return
	}

	// Need the session with the user ID.
	userID := h.connectUserID(r)
	if userID == model.InvalidUserID {
		http.Error(w, "No session found", http.StatusUnauthorized)
		return
	}
	dbToken, err := h.context.Storage.Token.Get(token)

	if err == storage.ErrNotFound {
		http.Error(w, "Unknown token", http.StatusNotFound)
		return
	}

	// If this is someone else's token - ignore it.
	if model.UserID(dbToken.UserID) != userID {
		http.Error(w, "Unknown token", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newTokenFromModel(dbToken)); err != nil {
			logging.Warning("Unable to marshal API token %s: %v", dbToken.Token, err)
		}

	case http.MethodPut:
		var values map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		resource, ok := values["resource"].(string)
		if ok {
			dbToken.Resource = resource
		}
		write, ok := values["write"].(bool)
		if ok {
			dbToken.Write = write
		}
		if !h.updateTags(&(dbToken.Tags), values) {
			http.Error(w, "Invalid tag name or value", http.StatusBadRequest)
			return
		}
		if err := h.context.Storage.Token.Update(dbToken, h.connectUserID(r)); err != nil {
			logging.Warning("Unable to update token: %v", err)
			http.Error(w, "Unable to update token", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newTokenFromModel(dbToken)); err != nil {
			logging.Warning("Unable to marshal token with user ID %v into JSON: %v", dbToken.UserID, err)
		}

	case http.MethodDelete:
		err := h.context.Storage.Token.Delete(token, userID)
		if err == storage.ErrNotFound {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}
		if err != nil {
			logging.Warning("Unable to remove token %s: %v", token, err)
			http.Error(w, "Unable to remove token", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

func (h *Server) listTokens(session goconnect.Session, w http.ResponseWriter, r *http.Request) {
	// Get the list of tokens for the user
	list, err := h.context.Storage.Token.GetList(h.connectUserID(r))
	if err != nil {
		logging.Warning("Unable to retrieve list of tokens: %v", err)
		http.Error(w, "Unable to retrieve list of tokens", http.StatusInternalServerError)
		return
	}
	tokens := apiTokenList{Tokens: make([]apiToken, 0)}
	for token := range list {
		tokens.Tokens = append(tokens.Tokens, newTokenFromModel(token))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(tokens); err != nil {
		logging.Warning("Unable to marshal token list to JSON: %v", err)
	}
}

func (h *Server) createToken(session goconnect.Session, w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logging.Info("Unable to read POSTed body to tokens: %v", err)
		http.Error(w, "Unable to read request body", http.StatusInternalServerError)
		return
	}
	newToken := apiToken{}
	if err = json.Unmarshal(buf, &newToken); err != nil {
		logging.Info("Invalid device JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(newToken.Resource) == "" {
		http.Error(w, "Must specify resource", http.StatusBadRequest)
		return
	}
	token, err := model.NewAPIToken(model.UserID(session.UserID), newToken.Resource, newToken.Write)
	if err != nil {
		logging.Warning("Unable to create token: %v", err)
		http.Error(w, "Unable to create token", http.StatusInternalServerError)
		return
	}

	if err := h.context.Storage.Token.Put(token, h.connectUserID(r)); err != nil {
		logging.Warning("Unable to store token %s: %v", token, err)
		http.Error(w, "Unable to store token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newTokenFromModel(token)); err != nil {
		logging.Warning("Unable to marshal token %s: %v", token, err)
	}
}

// tokenListHandler handles GET and POST requests to the token list
func (h *Server) tokenListHandler(w http.ResponseWriter, r *http.Request) {
	// Need the session with the user ID.
	s := r.Context().Value(goconnect.SessionContext)
	session, ok := s.(goconnect.Session)
	if !ok {
		http.Error(w, "No session found", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listTokens(session, w, r)
	case http.MethodPost:
		h.createToken(session, w, r)
	default:
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}
