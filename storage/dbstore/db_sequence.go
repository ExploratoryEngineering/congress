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
	"strings"

	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

// This will be the db-backed sequence implementation
type dbKeySequenceStorage struct {
	db              *sql.DB
	selectStatement *sql.Stmt
	updateStatement *sql.Stmt
	insertStatement *sql.Stmt
}

func (d *dbKeySequenceStorage) Close() {
	d.selectStatement.Close()
	d.updateStatement.Close()
	d.insertStatement.Close()
}

// NewDBKeySequenceStorage creates a new DB-backed KeySequenceStorage instance.
func NewDBKeySequenceStorage(db *sql.DB) (storage.KeySequenceStorage, error) {
	sqlSelect := `SELECT counter FROM lora_sequence WHERE identifier = $1 FOR UPDATE`
	sqlUpdate := `UPDATE lora_sequence SET counter = $1 WHERE identifier = $2`
	sqlInsert := `INSERT INTO lora_sequence (identifier, counter) VALUES ($1, $2)`

	ret := dbKeySequenceStorage{db: db}
	var err error
	if ret.selectStatement, err = db.Prepare(sqlSelect); err != nil {
		return nil, fmt.Errorf("unable to prepare select statement: %v", err)
	}
	if ret.insertStatement, err = db.Prepare(sqlInsert); err != nil {
		return nil, fmt.Errorf("unable to prepare insert statement: %v", err)
	}
	if ret.updateStatement, err = db.Prepare(sqlUpdate); err != nil {
		return nil, fmt.Errorf("unable to prepare update statement: %v", err)
	}

	return &ret, nil
}

func (d *dbKeySequenceStorage) AllocateKeys(identifier string, interval uint64, initial uint64) (chan uint64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	row := tx.Stmt(d.selectStatement).QueryRow(identifier)

	var start uint64
	var counter int64
	err = row.Scan(&counter)

	switch err {
	case sql.ErrNoRows:
		// Not found - insert a new one with interval prepopulated
		_, err = tx.Stmt(d.insertStatement).Exec(identifier, int64(initial+interval))
		if err != nil {
			tx.Rollback()
			if strings.Index(err.Error(), "lora_sequence_pk") >= 0 {
				// Retry since the key already exists
				return d.AllocateKeys(identifier, interval, initial)
			}
			return nil, err
		}
		counter = int64(initial)
	default:
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		_, err = tx.Stmt(d.updateStatement).Exec(counter+int64(interval), identifier)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		logging.Error("Unable to commit sequence with identifier %s (interval: %d, initial: %d): %v",
			identifier, interval, initial, err)
	}
	start = uint64(counter)

	ret := make(chan uint64)
	go func() {
		for i := start; i < (start + interval); i++ {
			ret <- i
		}
		close(ret)
	}()
	return ret, nil
}
