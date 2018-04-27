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
	"flag"
	"sync"
	"time"

	"github.com/ExploratoryEngineering/congress/storage/dbstore"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/logging"
)

var params struct {
	ConnectionString string
	UserCount        int
}

var defaultMA protocol.MA

func init() {
	flag.IntVar(&params.UserCount, "users", 1000, "Number of users to generate")
	flag.StringVar(&params.ConnectionString, "connectionstring", "postgres://localhost/congress?sslmode=disable", "PostgreSQL connection string")
	flag.Parse()

	var err error
	defaultMA, err = protocol.NewMA([]byte{0, 9, 9})
	if err != nil {
		panic(err)
	}
}

const tokensPerUser = 4
const appsPerUser = 5
const devicesPerApp = 30
const dataPerDevice = 100
const noncesPerDevice = 30
const gatewaysPerUser = 2
const outputsPerApp = 0

func main() {
	logging.EnableStderr(true)
	logging.SetLogLevel(logging.InfoLevel)
	logging.Info("This is the data generator tool")
	//datastore := memstore.CreateMemoryStorage(0, 0)
	datastore, err := dbstore.CreateStorage(params.ConnectionString, 100, 50, time.Minute*5)
	if err != nil {
		logging.Error("Unable to create datastore: %v", err)
		return
	}

	keygen, err := server.NewEUIKeyGenerator(defaultMA, 0, datastore.Sequence)
	if err != nil {
		logging.Error("Unable to create key generator: %v", err)
		return
	}

	const workers = 8
	wg := &sync.WaitGroup{}
	wg.Add(workers)

	messageChan := make(chan model.UserID)
	dataGenConsumer := func(ids chan model.UserID, wg *sync.WaitGroup) {
		for id := range ids {
			generateTokens(id, tokensPerUser, datastore)
			gateways := generateGateways(id, gatewaysPerUser, datastore)
			generateApplications(id, appsPerUser, datastore, &keygen, func(createdApplication model.Application) {
				generateOutputs(id, outputsPerApp, createdApplication)
				generateDevices(devicesPerApp, createdApplication, datastore, &keygen, func(createdDevice model.Device) {
					generateDeviceData(createdDevice, dataPerDevice, gateways, datastore)
					generateDownstreamMessage(createdDevice, datastore)
					generateNonces(createdDevice, noncesPerDevice, datastore)
				})
			})
		}
		wg.Done()
	}

	for i := 0; i < workers; i++ {
		go dataGenConsumer(messageChan, wg)
	}

	generateUsers(params.UserCount, datastore, &keygen, func(createdUser model.User) {
		messageChan <- createdUser.ID
	})

	close(messageChan)
	wg.Wait()
}
