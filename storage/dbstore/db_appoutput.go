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
	"encoding/json"
	"fmt"
	"time"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

type dbAppOutput struct {
	dbStore
	deleteStatement  *sql.Stmt
	appListStatement *sql.Stmt
	allListStatement *sql.Stmt
	putStatement     *sql.Stmt
	updateStatement  *sql.Stmt
}

func (d *dbAppOutput) Close() {
	d.deleteStatement.Close()
	d.appListStatement.Close()
	d.allListStatement.Close()
	d.putStatement.Close()
	d.updateStatement.Close()
}

// NewDBOutputStorage creates a new OutpubStorage instance backed by a
// database
func NewDBOutputStorage(db *sql.DB, userManagement storage.UserManagement) (storage.AppOutputStorage, error) {
	ret := dbAppOutput{dbStore{db: db, userManagement: userManagement}, nil, nil, nil, nil, nil}

	var err error
	sqlDelete := `DELETE FROM lora_output WHERE eui = $1`
	if ret.deleteStatement, err = db.Prepare(sqlDelete); err != nil {
		return nil, fmt.Errorf("unable to prepare delete statement: %v", err)
	}

	sqlAppList := `SELECT eui, config, application_eui FROM lora_output WHERE application_eui = $1`
	if ret.appListStatement, err = db.Prepare(sqlAppList); err != nil {
		return nil, fmt.Errorf("unable to prepare application statement: %v", err)
	}

	sqlAll := `SELECT eui, config, application_eui FROM lora_output`
	if ret.allListStatement, err = db.Prepare(sqlAll); err != nil {
		return nil, fmt.Errorf("unable to prepare all list statement: %v", err)
	}

	sqlInsert := `INSERT INTO lora_output (eui, config, application_eui) VALUES ($1, $2, $3)`
	if ret.putStatement, err = db.Prepare(sqlInsert); err != nil {
		return nil, fmt.Errorf("unable to prepare insert statement: %v", err)
	}

	sqlUpdate := `UPDATE lora_output SET config = $1 WHERE eui = $2`
	if ret.updateStatement, err = db.Prepare(sqlUpdate); err != nil {
		return nil, fmt.Errorf("unable to prepare update statement: %v", err)
	}

	return &ret, nil
}

func (d *dbAppOutput) Delete(op model.AppOutput) error {
	return d.doSQLExec(d.deleteStatement, func(s *sql.Stmt) (sql.Result, error) {
		return s.Exec(op.EUI.String())
	})
}

func (d *dbAppOutput) readOutput(rows *sql.Rows) (model.AppOutput, error) {
	var outputEUI, appEUI string
	var buf []uint8
	var err error
	op := model.AppOutput{}
	if err := rows.Scan(&outputEUI, &buf, &appEUI); err != nil {
		return op, err
	}
	if op.EUI, err = protocol.EUIFromString(outputEUI); err != nil {
		return op, err
	}
	if op.AppEUI, err = protocol.EUIFromString(appEUI); err != nil {
		return op, err
	}
	if err = json.Unmarshal(buf, &op.Configuration); err != nil {
		return op, err
	}
	return op, nil
}

func (d *dbAppOutput) getOutputList(rows *sql.Rows, err error) (<-chan model.AppOutput, error) {
	if err != nil {
		return nil, err
	}

	ret := make(chan model.AppOutput)
	go func() {
		defer rows.Close()
		defer close(ret)
		for rows.Next() {
			op, err := d.readOutput(rows)
			if err != nil {
				logging.Warning("Unable to read application output from storage: %v", err)
				continue
			}
			select {
			case ret <- op:
			// This is OK
			case <-time.After(1 * time.Second):
				continue

			}
		}
	}()
	return ret, nil
}

func (d *dbAppOutput) GetByApplication(appEUI protocol.EUI) (<-chan model.AppOutput, error) {
	return d.getOutputList(d.appListStatement.Query(appEUI.String()))
}

func (d *dbAppOutput) ListAll() (<-chan model.AppOutput, error) {
	return d.getOutputList(d.allListStatement.Query())
}

func (d *dbAppOutput) Put(op model.AppOutput) error {
	return d.doSQLExec(d.putStatement, func(s *sql.Stmt) (sql.Result, error) {
		buf, err := json.Marshal(op.Configuration)
		if err != nil {
			return nil, err
		}
		return s.Exec(op.EUI.String(), buf, op.AppEUI.String())
	})
}

func (d *dbAppOutput) Update(op model.AppOutput) error {
	return d.doSQLExec(d.updateStatement, func(s *sql.Stmt) (sql.Result, error) {
		buf, err := json.Marshal(op.Configuration)
		if err != nil {
			return nil, err
		}
		return s.Exec(buf, op.EUI.String())
	})
}
