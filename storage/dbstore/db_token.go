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
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

type dbTokenStore struct {
	dbStore
	selectStatement *sql.Stmt
	listStatement   *sql.Stmt
	deleteStatement *sql.Stmt
	insertStatement *sql.Stmt
	updateStatement *sql.Stmt
}

func (d *dbTokenStore) Close() {
	d.selectStatement.Close()
	d.listStatement.Close()
	d.deleteStatement.Close()
	d.insertStatement.Close()
	d.updateStatement.Close()
}

// NewDBTokenStorage creates a new database-backed token store
func NewDBTokenStorage(db *sql.DB, userManagement storage.UserManagement) (storage.TokenStorage, error) {
	ret := dbTokenStore{dbStore{db: db, userManagement: userManagement}, nil, nil, nil, nil, nil}

	var err error
	sqlSelect := `SELECT token, resource, write, user_id, tags
					FROM lora_token WHERE token = $1`
	if ret.selectStatement, err = db.Prepare(sqlSelect); err != nil {
		return nil, fmt.Errorf("unable to prepare select: %v", err)
	}
	sqlList := `SELECT token, resource, write, user_id, tags
					FROM lora_token WHERE user_id = $1`
	if ret.listStatement, err = db.Prepare(sqlList); err != nil {
		return nil, fmt.Errorf("unable to prepare list select: %v", err)
	}
	sqlDelete := `DELETE FROM lora_token WHERE token = $1 AND user_id = $2`
	if ret.deleteStatement, err = db.Prepare(sqlDelete); err != nil {
		return nil, fmt.Errorf("unable to prepare delete: %v", err)
	}
	sqlInsert := `INSERT INTO lora_token (token, resource, write, user_id, tags)
					VALUES ($1, $2, $3, $4, $5)`
	if ret.insertStatement, err = db.Prepare(sqlInsert); err != nil {
		return nil, fmt.Errorf("unable to prepare insert: %v", err)
	}
	sqlUpdate := `UPDATE lora_token
					SET resource = $1, write = $2, tags = $3
					WHERE token = $4`
	if ret.updateStatement, err = db.Prepare(sqlUpdate); err != nil {
		return nil, fmt.Errorf("unable to prepare update: %v", err)
	}
	return &ret, nil
}

func (d *dbTokenStore) Put(token model.APIToken, userID model.UserID) error {
	return d.doSQLExecWithOwner(d.insertStatement, func(s *sql.Stmt, ownerID uint64) (sql.Result, error) {
		return s.Exec(token.Token, token.Resource, token.Write, string(userID), token.TagJSON())
	}, userID)
}

func (d *dbTokenStore) Delete(token string, userID model.UserID) error {
	return d.doSQLExecWithOwner(d.deleteStatement, func(s *sql.Stmt, ownerID uint64) (sql.Result, error) {
		return s.Exec(token, string(userID))
	}, userID)
}

func (d *dbTokenStore) readToken(rows *sql.Rows) (model.APIToken, error) {
	ret := model.APIToken{}
	var tagBuffer []byte
	if err := rows.Scan(&ret.Token, &ret.Resource, &ret.Write, &ret.UserID, &tagBuffer); err != nil {
		return ret, err
	}
	tags, err := model.NewTagsFromBuffer(tagBuffer)
	if err != nil {
		return ret, fmt.Errorf("invalid tag buffer for token: %v (userID = %v)", err, ret.UserID)
	}
	ret.Tags = *tags
	return ret, nil
}

func (d *dbTokenStore) GetList(userID model.UserID) (chan model.APIToken, error) {
	uid := string(userID)
	rows, err := d.listStatement.Query(uid)
	if err != nil {
		return nil, err
	}
	ret := make(chan model.APIToken)
	go func() {
		defer rows.Close()
		defer close(ret)
		for rows.Next() {
			token, err := d.readToken(rows)
			if err != nil {
				logging.Warning("Unable to read token: %v", err)
				continue
			}
			ret <- token
		}
	}()
	return ret, nil
}

func (d *dbTokenStore) Get(token string) (model.APIToken, error) {
	rows, err := d.selectStatement.Query(token)
	if err != nil {
		return model.APIToken{}, err
	}

	defer rows.Close()

	if !rows.Next() {
		return model.APIToken{}, storage.ErrNotFound
	}

	return d.readToken(rows)
}

func (d *dbTokenStore) Update(token model.APIToken, userID model.UserID) error {
	return d.doSQLExecWithOwner(d.updateStatement, func(s *sql.Stmt, ownerID uint64) (sql.Result, error) {
		return s.Exec(token.Resource, token.Write, token.Tags.TagJSON(), token.Token)
	}, userID)
}
