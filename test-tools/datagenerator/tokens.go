package main

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
	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

func generateTokens(id model.UserID, count int, datastore storage.Storage) {
	for i := 0; i < count; i++ {
		randomToken, err := model.NewAPIToken(id, "/", true)
		if err != nil {
			logging.Error("Unable to create token: %v", err)
		} else {
			if err := datastore.Token.Put(randomToken, id); err != nil {
				logging.Error("Unable to store token: %v", err)
			}
		}
	}
}
