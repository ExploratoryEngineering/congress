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
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
	// Use the Postgres driver
	_ "github.com/lib/pq"
)

// CreateSchema crreates the schema for the database
func CreateSchema(db *sql.DB) {
	commands := SchemaCommandList()
	for _, v := range commands {
		if _, err := db.Exec(v); err != nil {
			msg := fmt.Sprintf("Unable to create PostgreSQL schema: %v (while running %s)", err, v)
			panic(msg)
		}
	}
	logging.Info("PostgreSQL schema created")
}

// dbStore is the base type for all the backend storage implementations
type dbStore struct {
	db             *sql.DB
	userManagement storage.UserManagement
}

// putFunc is a function used by the dbSQLExec wrappers
type stmtOwnerFunc func(stmt *sql.Stmt, ownerID uint64) (sql.Result, error)
type stmtFunc func(stmt *sql.Stmt) (sql.Result, error)

func (d *dbStore) doSQLExec(statement *sql.Stmt, execFunc stmtFunc) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	stmt := tx.Stmt(statement)
	var result sql.Result
	if result, err = execFunc(stmt); err != nil {
		tx.Rollback()
		errMsg := err.Error()
		if strings.Index(errMsg, "duplicate key value violates") > 0 {
			return storage.ErrAlreadyExists
		}
		if strings.Index(errMsg, "violates foreign key constraint") > 0 {
			return storage.ErrDeleteConstraint
		}
		return err
	}
	tx.Commit()
	if count, _ := result.RowsAffected(); count == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// doSQLExec wraps an Exec statement in a transaction and returns the proper error
func (d *dbStore) doSQLExecWithOwner(statement *sql.Stmt, execFunc stmtOwnerFunc, userID model.UserID) error {
	ownerID, err := d.userManagement.GetOwner(userID)
	if err != nil {
		logging.Warning("Unable to get owner ID for user with ID %s: %v", string(userID), err)
		return err
	}

	return d.doSQLExec(statement, func(s *sql.Stmt) (sql.Result, error) {
		return execFunc(s, ownerID)
	})
}

// CreateStorage creates a new storage
func CreateStorage(connectionString string, maxConn, idleConn int, maxConnLifetime time.Duration) (storage.Storage, error) {
	db, err := sql.Open("postgres", connectionString)
	if nil != err {
		log.Fatal(fmt.Sprintf("Unable to connect to database: %s", err))
		return storage.Storage{}, err
	}
	db.SetMaxIdleConns(idleConn)
	db.SetMaxOpenConns(maxConn)
	db.SetConnMaxLifetime(maxConnLifetime)

	userManagement, err := NewDBUserManagement(db)

	var appStorage storage.ApplicationStorage
	if appStorage, err = NewDBApplicationStorage(db, userManagement); err != nil {
		return storage.Storage{}, fmt.Errorf("unable to create application storage: %v", err)
	}

	var devStorage storage.DeviceStorage
	if devStorage, err = NewDBDeviceStorage(db, userManagement); err != nil {
		return storage.Storage{}, fmt.Errorf("unable to create device storage: %v", err)
	}

	var dataStorage storage.DataStorage
	if dataStorage, err = NewDBDataStorage(db, userManagement); err != nil {
		return storage.Storage{}, fmt.Errorf("unable to create data storage: %v", err)
	}

	var sequenceStorage storage.KeySequenceStorage
	if sequenceStorage, err = NewDBKeySequenceStorage(db); err != nil {
		return storage.Storage{}, fmt.Errorf("unable to create sequence storage: %v", err)
	}

	var gatewayStorage storage.GatewayStorage
	if gatewayStorage, err = NewDBGatewayStorage(db, userManagement); err != nil {
		return storage.Storage{}, fmt.Errorf("unable to create gateway storage: %v", err)
	}

	var tokenStorage storage.TokenStorage
	if tokenStorage, err = NewDBTokenStorage(db, userManagement); err != nil {
		return storage.Storage{}, fmt.Errorf("unable to create token storage: %v", err)
	}

	var outputStorage storage.AppOutputStorage
	if outputStorage, err = NewDBOutputStorage(db, userManagement); err != nil {
		return storage.Storage{}, fmt.Errorf("unable to create output storage: %v", err)
	}
	return storage.Storage{
		Application:    appStorage,
		Device:         devStorage,
		DeviceData:     dataStorage,
		Sequence:       sequenceStorage,
		Gateway:        gatewayStorage,
		Token:          tokenStorage,
		UserManagement: userManagement,
		AppOutput:      outputStorage}, nil

}
