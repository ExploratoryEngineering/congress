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
	"math"
	"math/rand"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

func generateGateways(id model.UserID, count int, datastore storage.Storage) []model.Gateway {
	var gws []model.Gateway
	for i := 0; i < count; i++ {
		newGW := model.NewGateway()
		newGW.SetTag("name", fmt.Sprintf("Number %d", i))
		newGW.Latitude = float32(rand.Intn(180)) / math.Pi
		newGW.Longitude = float32(rand.Intn(360)) / math.Pi
		newGW.Altitude = 1.0
		newGW.GatewayEUI = randomEUI()
		newGW.IP = randomIP()
		newGW.StrictIP = rand.Int()%2 == 0
		if err := datastore.Gateway.Put(newGW, id); err != nil {
			logging.Error("Unable to store gateway: %v", err)
		} else {
			gws = append(gws, newGW)
		}
	}
	return gws
}
