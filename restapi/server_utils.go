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
	"fmt"
	"net/http"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/rest"
	"github.com/telenordigital/goconnect"
)

// Extract EUI from path parameter in context
func euiFromPathParameter(r *http.Request, name string) (protocol.EUI, error) {
	p := r.Context().Value(rest.PathParameter(name))
	euiStr, ok := p.(string)
	if !ok {
		return protocol.EUI{}, fmt.Errorf("no parameter named %s in request context", name)
	}
	eui, err := protocol.EUIFromString(euiStr)
	if err != nil {
		return protocol.EUI{}, err
	}
	return eui, nil
}

func newUserFromSession(session goconnect.Session) model.User {
	ret := model.User{
		ID:    model.UserID(session.UserID),
		Name:  session.Name,
		Email: session.Email,
	}
	if !session.VerifiedEmail {
		ret.Email = ""
	}
	return ret
}
