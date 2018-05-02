package storagetest

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
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/storage"
)

func testOutputStorage(s *storage.Storage, t *testing.T) {
	if s.AppOutput == nil {
		t.Fatal("Output isn't set for storage")
	}

	application := model.NewApplication()
	application.AppEUI = makeRandomEUI()
	s.Application.Put(application, model.SystemUserID)

	configOne, _ := model.NewTransportConfig(`{"username": "user1"}`)
	opOne := model.AppOutput{EUI: makeRandomEUI(), AppEUI: application.AppEUI, Configuration: configOne}
	if err := s.AppOutput.Put(opOne); err != nil {
		t.Fatal("Got error storing output: ", err)
	}

	configTwo, _ := model.NewTransportConfig(`{}`)
	opTwo := model.AppOutput{EUI: makeRandomEUI(), AppEUI: application.AppEUI, Configuration: configTwo}
	if err := s.AppOutput.Put(opTwo); err != nil {
		t.Fatal("Got error storing 2nd output: ", err)
	}

	// update the 2nd configuration
	configTwo[model.TransportConfigKey("username")] = "user2"
	if err := s.AppOutput.Update(opTwo); err != nil {
		t.Fatal("Couldn't update config: ", err)
	}

	ch, err := s.AppOutput.GetByApplication(application.AppEUI)
	if err != nil {
		t.Fatal("Got error retrieving configurations: ", err)
	}

	count := 0
	for v := range ch {
		if v.EUI != opOne.EUI && v.EUI != opTwo.EUI {
			t.Fatal("Found unknown config: ", v)
		}
		count++
	}
	if count != 2 {
		t.Fatal("Expected just 2 outputs")
	}

	// Remove one of the outputs
	if err := s.AppOutput.Delete(opTwo); err != nil {
		t.Fatal("Got error deleting output #2: ", err)
	}

	if err := s.AppOutput.Delete(opOne); err != nil {
		t.Fatal("Got error deleting output #1: ", err)
	}

	configThree, _ := model.NewTransportConfig("")
	opThree := model.AppOutput{EUI: makeRandomEUI(), AppEUI: application.AppEUI, Configuration: configThree}
	s.AppOutput.Put(opThree)

	ch, err = s.AppOutput.ListAll()
	if err != nil {
		t.Fatal("Got error listing outputs: ", err)
	}

	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No output on channel")
	}

	select {
	case op, ok := <-ch:
		if ok {
			t.Fatalf("Did not expect two items on channel but got %v", op)
		}
	default:
		t.Fatal("No output on channel")
	}
}
