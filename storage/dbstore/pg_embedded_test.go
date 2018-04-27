package dbstore

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

	"github.com/ExploratoryEngineering/logging"
)

// Test the embedded PostgreSQL database
func TestLaunchEmbedded(t *testing.T) {
	if !checkPostgresInstallation() {
		logging.Error("PostgreSQL not found on system. The test won't run")
		logging.Error("This will affect coverage for the various unit tests")
		t.Skip("PostgreSQL isn't installed")
	}
	tempDir, err := getDBTempDir()
	if err != nil {
		t.Fatal(err)
	}
	pg := newPostgresEmbedded(tempDir)

	pg.InitializeNew()
	err = pg.Start()
	if err != nil {
		t.Fatal("Error launching: ", err)
	}

	pg.Stop()
}
