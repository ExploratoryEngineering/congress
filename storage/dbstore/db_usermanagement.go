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

const ownerKeyName string = "ownerid"

type dbUserManagement struct {
	db                    *sql.DB
	getUserStatement      *sql.Stmt
	updateUserStatement   *sql.Stmt
	insertUserStatement   *sql.Stmt
	getUserOwnerStatement *sql.Stmt
	insertOwnerStatement  *sql.Stmt
}

func (d *dbUserManagement) Close() {
	d.getUserStatement.Close()
	d.updateUserStatement.Close()
	d.insertUserStatement.Close()
	d.getUserOwnerStatement.Close()
	d.insertOwnerStatement.Close()
}

// NewDBUserManagement creates a new database-backed UserManagement instance
func NewDBUserManagement(db *sql.DB) (storage.UserManagement, error) {
	ret := dbUserManagement{
		db: db,
	}
	var err error

	getUserQuery := `SELECT user_id, name, email FROM lora_user WHERE user_id = $1`
	if ret.getUserStatement, err = db.Prepare(getUserQuery); err != nil {
		return nil, err
	}
	updateUserQuery := `UPDATE lora_user SET name = $1, email = $2 WHERE user_id = $3`
	if ret.updateUserStatement, err = db.Prepare(updateUserQuery); err != nil {
		return nil, err
	}
	insertUserQuery := `INSERT INTO lora_user (user_id, name, email) VALUES ($1, $2, $3)`
	if ret.insertUserStatement, err = db.Prepare(insertUserQuery); err != nil {
		return nil, err
	}

	getOwnerQuery := `SELECT owner_id FROM lora_owner WHERE user_id = $1`
	if ret.getUserOwnerStatement, err = db.Prepare(getOwnerQuery); err != nil {
		return nil, err
	}
	insertOwnerQuery := `INSERT INTO lora_owner (owner_id, user_id, org_id) VALUES ($1, $2, $3)`
	if ret.insertOwnerStatement, err = db.Prepare(insertOwnerQuery); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (d *dbUserManagement) createUser(tx *sql.Tx, userID string, name string, email string) error {
	// Nothing found - insert a new
	s := tx.Stmt(d.insertUserStatement)
	if _, err := s.Exec(userID, name, email); err != nil {
		logging.Error("Unable to insert user with ID %s: %v", userID, err)
		s.Close()
		tx.Rollback()
		return err
	}
	s.Close()
	return nil
}

func (d *dbUserManagement) updateUser(userID string, name string, email string) error {
	tx, err := d.db.Begin()
	if err != nil {
		logging.Error("Unable to start transaction for user update (ID:%s): %v", userID, err)
		return err
	}
	updateStmt := tx.Stmt(d.updateUserStatement)
	defer updateStmt.Close()
	if _, err := updateStmt.Exec(name, email, userID); err != nil {
		logging.Error("Unable to update user with ID %s: %v", userID, err)
		updateStmt.Close()
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (d *dbUserManagement) AddOrUpdateUser(user model.User, keyGen storage.KeyGeneratorFunc) error {
	// Look up the user.
	rows, err := d.getUserStatement.Query(string(user.ID))
	if err != nil {
		logging.Warning("Unable to retrieve user with ID %v: %v", user.ID, err)
		return err
	}
	if !rows.Next() {
		rows.Close()
		tx, err := d.db.Begin()
		if err != nil {
			logging.Error("Unable to start transaction for user %v: %v", user.ID, err)
			return err
		}
		if err := d.createUser(tx, string(user.ID), user.Name, user.Email); err != nil {
			return err
		}
		// Insert an entry into the owner table as well
		ownerStmt := tx.Stmt(d.insertOwnerStatement)
		defer ownerStmt.Close()
		if _, err := ownerStmt.Exec(keyGen(ownerKeyName), string(user.ID), sql.NullInt64{Valid: false}); err != nil {
			logging.Error("Unable to insert owner for user %v: %v", user.ID, err)
			tx.Rollback()
			return err
		}
		// Invariant: User is inserted. Commit and return
		tx.Commit()
		return nil
	}
	defer rows.Close()
	// Invariant: It already exists. Retrieve values and compare
	var userID, name, email string
	if err := rows.Scan(&userID, &name, &email); err != nil {
		logging.Warning("Unable to read fields on user with ID %v: %v", user.ID, err)
		return err
	}
	if user.Name != name || user.Email != email {
		// Invariant: Fields are different: Update with new
		// values.
		return d.updateUser(string(user.ID), user.Name, user.Email)
	}
	// Invariant: User is identical. Roll back transaction and return.
	return nil
}

func (d *dbUserManagement) GetOwner(userID model.UserID) (uint64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		logging.Error("Unable to create transaction for user with ID %v: %v", userID, err)
		return 0, err
	}

	s := tx.Stmt(d.getUserOwnerStatement)
	rows, err := s.Query(string(userID))
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	defer rows.Close()
	defer tx.Rollback()
	if !rows.Next() {
		// Invariant: No rows returned. Create a new owner entry
		return 0, fmt.Errorf("unable to find owner for user ID '%v'", userID)
	}

	var existingOwnerID int64
	// Invariant: Owner exists. Return the value
	if err := rows.Scan(&existingOwnerID); err != nil {
		logging.Warning("Owner query failed for user with ID %v: %v", existingOwnerID, err)
		return 0, err
	}

	return uint64(existingOwnerID), nil
}
