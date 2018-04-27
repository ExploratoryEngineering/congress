package server

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
	"crypto/rand"
	"testing"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage/memstore"
	"github.com/ExploratoryEngineering/pubsub"
)

func makeRandomEUI() protocol.EUI {
	ret := protocol.EUI{}
	rand.Read(ret.Octets[:])
	return ret
}

func makeRandomOutput() model.AppOutput {
	op1 := model.NewAppOutput()
	op1.EUI = makeRandomEUI()
	op1.AppEUI = makeRandomEUI()
	op1.Configuration = map[model.TransportConfigKey]interface{}{
		"type": "log",
	}
	return op1
}

func TestAppOutputManager(t *testing.T) {
	// Create a list of outputs
	outputStorage := memstore.NewMemoryOutput()

	var opList []model.AppOutput
	for i := 0; i < 10; i++ {
		op := makeRandomOutput()
		if err := outputStorage.Put(op); err != nil {
			t.Fatal("Couldn't store output: ", err)
		}
		opList = append(opList, op)
	}

	router := pubsub.NewEventRouter(5)
	appMgr := NewAppOutputManager(&router)

	appMgr.LoadOutputs(outputStorage)
	defer appMgr.Shutdown()

	// Add two outputs
	op1 := makeRandomOutput()
	if err := appMgr.Add(&op1); err != nil {
		t.Fatal("Got error adding output #1: ", err)
	}

	op2 := makeRandomOutput()
	if err := appMgr.Add(&op2); err != nil {
		t.Fatal("Got error adding output #2: ", err)
	}

	// Try removing unknown output
	opUnknown := makeRandomOutput()
	if err := appMgr.Remove(&opUnknown); err != ErrNotFound {
		t.Fatal("Expected not found error but got ", err)
	}

	// Remove the first five outputs, update the last
	for i := 0; i < 5; i++ {
		op := opList[i]
		if err := appMgr.Remove(&op); err != nil {
			t.Fatalf("Got error removing output with EUI %s: %v", op.EUI, err)
		}
	}
	opList = opList[5:]
	for _, v := range opList {
		v.Configuration[model.TransportConfigKey("foo")] = "bar"
		if err := appMgr.Update(&v); err != nil {
			t.Fatalf("Got error updating output with EUI %s: %v", v.EUI, err)
		}
	}

	// Get status for the remaining outputs. It should be "running"
	for _, v := range opList {
		status, logs, err := appMgr.GetStatusAndLogs(&v)
		if err != nil {
			t.Fatal("Got error retrieving status for output: ", err)
		}
		if status != "idle" {
			t.Fatalf("Expected status 'idle' but got '%s' for output with EUI %s", status, v.EUI)
		}
		if logs == nil {
			t.Fatalf("nil logs for output with EUI %s", v.EUI)
		}
	}

	opList = append(opList, op1)
	for i, v := range opList {
		if err := appMgr.Remove(&v); err != nil {
			t.Fatalf("Got error removing op #%d: %v", i, err)
		}
	}

	// op2 should be shut down on exit
}
