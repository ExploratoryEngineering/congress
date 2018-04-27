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

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

// dbApplicationStorage implements application storage for PostgreSQL.
type dbApplicationStorage struct {
	dbStore
	putStatement       *sql.Stmt // Prepared statement for Put
	getStatement       *sql.Stmt // Prepared statement for GetByEUI
	listStatement      *sql.Stmt // Prepared statement for GetByNetworkEUI
	deleteStatement    *sql.Stmt // Prepared statement for Delete
	systemGetStatement *sql.Stmt // Prepared statement for system get
	updateStatement    *sql.Stmt // Prepared statement app update
}

// Close releases all of the resources used by the application storage.
func (d *dbApplicationStorage) Close() {
	d.putStatement.Close()
	d.getStatement.Close()
	d.listStatement.Close()
	d.deleteStatement.Close()
	d.systemGetStatement.Close()
	d.updateStatement.Close()
}

// NewDBApplicationStorage creates a new ApplicationStorage instance for
// PostgreSQL backends
func NewDBApplicationStorage(db *sql.DB, userManagement storage.UserManagement) (storage.ApplicationStorage, error) {
	var err error
	ret := dbApplicationStorage{dbStore{db: db, userManagement: userManagement}, nil, nil, nil, nil, nil, nil}

	sqlInsert := `
		INSERT INTO
			lora_application (
				eui,
				owner_id,
				tags)
		VALUES (
			$1,
			$2,
			$3)`
	if ret.putStatement, err = db.Prepare(sqlInsert); err != nil {
		return nil, fmt.Errorf("unable to prepare insert statement: %v", err)
	}

	sqlSelect := `
		SELECT
			a.eui,
			a.tags
		FROM
			lora_application a,
			lora_owner o
		WHERE
			a.eui = $1 AND a.owner_id = o.owner_id AND o.user_id = $2`
	if ret.getStatement, err = db.Prepare(sqlSelect); err != nil {
		return nil, fmt.Errorf("unable to prepare select statement: %v", err)
	}

	sqlList := `
		SELECT
			a.eui,
			a.tags
		FROM
			lora_application a, lora_owner o
		WHERE
			a.owner_id = o.owner_id AND o.user_id = $1`

	if ret.listStatement, err = db.Prepare(sqlList); err != nil {
		return nil, fmt.Errorf("unable to prepare list statement: %v", err)
	}

	sqlDelete := `
		DELETE
		FROM lora_application a
		USING lora_owner o
		WHERE a.eui = $1 AND a.owner_id = o.owner_id AND o.user_id = $2`
	if ret.deleteStatement, err = db.Prepare(sqlDelete); err != nil {
		return nil, fmt.Errorf("unable to prepare delete statement: %v", err)
	}

	sqlSystemGet := `
		SELECT
			a.eui,
			a.tags
		FROM
			lora_application a
		WHERE
			a.eui = $1`
	if ret.systemGetStatement, err = db.Prepare(sqlSystemGet); err != nil {
		return nil, fmt.Errorf("unable to prepare system select statement: %v", err)
	}

	sqlUpdate := `
		UPDATE
			lora_application a
		SET
			tags = $1
		FROM
			lora_owner o
		WHERE
			a.eui = $2 AND a.owner_id = o.owner_id AND o.user_id = $3`
	if ret.updateStatement, err = db.Prepare(sqlUpdate); err != nil {
		return nil, fmt.Errorf("unable to prepare app update statement: %v", err)
	}
	return &ret, nil
}

func (d *dbApplicationStorage) readApplication(rows *sql.Rows) (model.Application, error) {
	var appEUI string
	var err error
	var tagBuffer []byte
	ret := model.NewApplication()
	if err = rows.Scan(&appEUI, &tagBuffer); err != nil {
		return ret, err
	}

	if ret.AppEUI, err = protocol.EUIFromString(appEUI); err != nil {
		return ret, fmt.Errorf("invalid App EUI for application: %v (eui=%s)", err, appEUI)
	}

	tags, err := model.NewTagsFromBuffer(tagBuffer)
	if err != nil {
		return ret, fmt.Errorf("invalid tag buffer for application: %v (eui=%s)", err, appEUI)
	}
	ret.Tags = *tags
	return ret, nil
}

// GetByEUI retrieves the application with the specified application EUI.
func (d *dbApplicationStorage) GetByEUI(eui protocol.EUI, userID model.UserID) (model.Application, error) {
	var rows *sql.Rows
	var err error
	if userID == model.SystemUserID {
		rows, err = d.systemGetStatement.Query(eui.String())
	} else {
		rows, err = d.getStatement.Query(eui.String(), string(userID))
	}
	ret := model.NewApplication()
	if err != nil {
		return ret, err
	}
	defer rows.Close()
	if !rows.Next() {
		return ret, storage.ErrNotFound
	}
	app, err := d.readApplication(rows)
	return app, err
}

// GetByNetworkEUI returns all applications with the given network EUI
func (d *dbApplicationStorage) GetList(userID model.UserID) (chan model.Application, error) {
	rows, err := d.listStatement.Query(string(userID))
	if err != nil {
		return nil, fmt.Errorf("unable to query application list: %v", err)
	}
	outputChan := make(chan model.Application)
	go func() {
		defer rows.Close()
		defer close(outputChan)
		for rows.Next() {
			app, err := d.readApplication(rows)
			if err != nil {
				logging.Warning("Unable to read application in list, skipping: %v", err)
				continue
			}
			outputChan <- app
		}
	}()
	return outputChan, nil
}

// Put stores an Application instance in the storage backend
func (d *dbApplicationStorage) Put(application model.Application, userID model.UserID) error {
	return d.doSQLExecWithOwner(d.putStatement, func(s *sql.Stmt, ownerID uint64) (sql.Result, error) {
		return s.Exec(application.AppEUI.String(),
			ownerID,
			application.Tags.TagJSON())
	}, userID)
}

func (d *dbApplicationStorage) Delete(eui protocol.EUI, userID model.UserID) error {
	return d.doSQLExecWithOwner(d.deleteStatement, func(s *sql.Stmt, ownerID uint64) (sql.Result, error) {
		return s.Exec(eui.String(), string(userID))
	}, userID)
}

func (d *dbApplicationStorage) Update(application model.Application, userID model.UserID) error {
	tagBuffer := application.TagJSON()
	return d.doSQLExecWithOwner(d.updateStatement, func(s *sql.Stmt, ownerID uint64) (sql.Result, error) {
		return s.Exec(tagBuffer, application.AppEUI.String(), string(userID))
	}, userID)
}
