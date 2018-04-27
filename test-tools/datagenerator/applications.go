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
	"fmt"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

func generateApplications(id model.UserID, count int, datastore storage.Storage, keyGen *server.KeyGenerator, callback func(generatedApp model.Application)) {
	for i := 0; i < count; i++ {
		app := model.NewApplication()
		var err error
		app.AppEUI, err = keyGen.NewAppEUI()
		if err != nil {
			logging.Error("Unable to generate app EUI. Using random EUI")
			app.AppEUI = randomEUI()
		}
		app.SetTag("name", fmt.Sprintf("App %d owned by %s", i, id))
		if err := datastore.Application.Put(app, id); err != nil {
			logging.Error("Unable to store application: %v", err)
		} else {
			callback(app)
		}
	}
}

func generateOutputs(id model.UserID, count int, app model.Application) {
	// Skip this ... for now. It causes lots of issues if we try to attack the
	// internet with MQTT broker lookups. Or ourselves.
}
