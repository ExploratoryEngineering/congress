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
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/congress/storage/storagetest"
	"github.com/ExploratoryEngineering/logging"
)

var pgdb *postgresEmbedded
var db *sql.DB
var userManagement storage.UserManagement

func TestMain(m *testing.M) {
	if !checkPostgresInstallation() {
		logging.Error("**** PostgreSQL not installed")
		os.Exit(0)
	}

	pgdb, db = createSchemaAndDB()

	var err error
	userManagement, err = NewDBUserManagement(db)
	if err != nil {
		logging.Error("Error creating people store: %v", err)
		os.Exit(1)
	}
	// call flag.Parse() here if TestMain uses flags
	ret := m.Run()
	db.Close()
	pgdb.Stop()
	os.Exit(ret)
}

func createSchemaAndDB() (*postgresEmbedded, *sql.DB) {
	tempDir, err := getDBTempDir()
	if err != nil {
		logging.Error("Can't get temp dir: %v", err)
		os.Exit(1)
	}
	pgdb := newPostgresEmbedded(tempDir)
	pgdb.InitializeNew()
	pgdb.Start()
	if err != nil {
		logging.Error("Error starting PostgreSQL: %v", err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", pgdb.GetConnectionOptions())
	if err != nil {
		logging.Error("Error loading driver: %v", err)
		os.Exit(1)
	}
	if err = db.Ping(); err != nil {
		pgdb.Stop()
		db.Close()
		logging.Error("Couldn't ping the database: %v", err)
		os.Exit(1)
	}
	CreateSchema(db)
	return pgdb, db
}

func TestDBStorage(t *testing.T) {

	storage, err := CreateStorage(pgdb.GetConnectionOptions(), 10, 5, time.Minute)
	if err != nil {
		t.Fatalf("Couldn't create db storage: %v", err)
	}
	defer storage.Close()
	storagetest.DoStorageTests(&storage, t)

}
