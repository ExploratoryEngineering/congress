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

func testApplicationStorage(
	appStorage storage.ApplicationStorage,
	userID model.UserID,
	t *testing.T) {

	application := model.Application{
		AppEUI: makeRandomEUI(),
		Tags:   model.NewTags(),
	}

	if err := appStorage.Put(application, userID); err != nil {
		t.Error("Got error adding application: ", err)
	}

	// Rinse and repeat
	if err := appStorage.Put(application, userID); err == nil {
		t.Error("Shouldn't be able to add application twice: ", err)
	}

	// Open the application
	existingApp, err := appStorage.GetByEUI(application.AppEUI, userID)
	if err != nil {
		t.Error("Shouldn't get error when opening an application that is added: ", err)
	}
	if !existingApp.Equals(application) {
		t.Error("The application doesn't match the stored one")
	}
	// Try to open an application that doesn't exist
	if _, err = appStorage.GetByEUI(makeRandomEUI(), userID); err == nil {
		t.Error("Shouldn't be able to open unknown application")
	}

	// Get list of all applications
	found := 0
	appCh, err := appStorage.GetList(userID)
	select {
	case <-appCh:
		found++
	case <-time.After(time.Millisecond * 100):
		t.Error("Did not get any data on app channel")
	}

	if found == 0 {
		t.Error("Did not get any data on app channel")
	}

	application.SetTag("Updated Tag", "With Something")
	if err := appStorage.Update(application, userID); err != nil {
		t.Fatalf("Got error updating tags for application: %v", err)
	}

	// Update application
	application.Tags.SetTag("Foo", "Bar")
	if err := appStorage.Update(application, userID); err != nil {
		t.Fatalf("Couldn't update app: %v", err)
	}

	// Use another User ID -- it should return not found
	if _, err := appStorage.GetByEUI(application.AppEUI, model.UserID("foo")); err != storage.ErrNotFound {
		t.Fatalf("App should not be visible to another user but error returned wasn't ErrNotFound (%v)", err)
	}

	updatedApp, _ := appStorage.GetByEUI(application.AppEUI, userID)
	foo1, _ := updatedApp.Tags.GetTag("foo")
	foo2, _ := application.Tags.GetTag("foo")
	if foo1 != foo2 {
		t.Fatalf("App isn't updated properly. Updated app is %v but should be %v", updatedApp, application)
	}

	// Update app that doesn't exist
	unknownApp := model.NewApplication()
	unknownApp.AppEUI = makeRandomEUI()
	if err := appStorage.Update(unknownApp, userID); err != storage.ErrNotFound {
		t.Fatalf("Expected update to fail for unknown app but it succeeded")
	}

	if err := appStorage.Delete(application.AppEUI, userID); err != nil {
		t.Fatalf("Got error deleting application: %v", err)
	}

	if err := appStorage.Delete(application.AppEUI, userID); err == nil {
		t.Fatal("Expected error when deleting application but didn't get one")
	}
}
