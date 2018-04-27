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

// sbDeviceStorage is a device storage that stores data in PostgreSQL
type dbDeviceStorage struct {
	dbStore
	putStatement         *sql.Stmt
	devAddrStatement     *sql.Stmt
	euiStatement         *sql.Stmt
	nonceStatement       *sql.Stmt
	appEUIStatement      *sql.Stmt
	getNonceStatement    *sql.Stmt
	updateStateStatement *sql.Stmt
	deleteStatement      *sql.Stmt
	updateStatement      *sql.Stmt
}

// Close closes all resources allocated by the DBDeviceStorage instance.
func (d *dbDeviceStorage) Close() {
	d.putStatement.Close()
	d.devAddrStatement.Close()
	d.euiStatement.Close()
	d.nonceStatement.Close()
	d.appEUIStatement.Close()
	d.getNonceStatement.Close()
	d.updateStateStatement.Close()
	d.deleteStatement.Close()
	d.updateStatement.Close()
}

// NewDBDeviceStorage returns a new PostgreSQL-backed device storage
func NewDBDeviceStorage(db *sql.DB, userManagement storage.UserManagement) (storage.DeviceStorage, error) {
	ret := dbDeviceStorage{dbStore{db: db, userManagement: userManagement}, nil, nil, nil, nil, nil, nil, nil, nil, nil}
	var err error

	sqlInsert := `
		INSERT INTO
			lora_device (
				eui,
				dev_addr,
				app_key,
				apps_key,
				nwks_key,
				application_eui,
				state,
				fcnt_up,
				fcnt_dn,
				relaxed_counter,
				key_warning,
				tags)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12)`
	if ret.putStatement, err = db.Prepare(sqlInsert); err != nil {
		return nil, fmt.Errorf("unable to prepare insert statement: %v", err)
	}

	sqlSelect := `
		SELECT
			eui,
			dev_addr,
			app_key,
			apps_key,
			nwks_key,
			application_eui,
			state,
			fcnt_up,
			fcnt_dn,
			relaxed_counter,
			key_warning,
			tags
		FROM
			lora_device
		WHERE
			dev_addr = $1`
	if ret.devAddrStatement, err = db.Prepare(sqlSelect); err != nil {
		return nil, fmt.Errorf("unable to prepare select statement: %v", err)
	}

	sqlList := `
		SELECT
			eui,
			dev_addr,
			app_key,
			apps_key,
			nwks_key,
			application_eui,
			state,
			fcnt_up,
			fcnt_dn,
			relaxed_counter,
			key_warning,
			tags
		FROM
			lora_device
		WHERE
			application_eui = $1`

	if ret.appEUIStatement, err = db.Prepare(sqlList); err != nil {
		return nil, fmt.Errorf("unable to prepare list statement: %v", err)
	}

	euiSelect := `
		SELECT
			eui,
			dev_addr,
			app_key,
			apps_key,
			nwks_key,
			application_eui,
			state,
			fcnt_up,
			fcnt_dn,
			relaxed_counter,
			key_warning,
			tags
		FROM
			lora_device
		WHERE
			eui = $1`

	if ret.euiStatement, err = db.Prepare(euiSelect); err != nil {
		return nil, fmt.Errorf("unable to prepare eui select statement: %v", err)
	}

	nonceInsert := `INSERT INTO lora_device_nonce (device_eui, nonce) VALUES ($1, $2)`
	if ret.nonceStatement, err = db.Prepare(nonceInsert); err != nil {
		return nil, fmt.Errorf("unable to prepare nonce insert statement: %v", err)
	}

	nonceSelect := `SELECT nonce FROM lora_device_nonce WHERE device_eui = $1`
	if ret.getNonceStatement, err = db.Prepare(nonceSelect); err != nil {
		return nil, fmt.Errorf("unable to prepare nonce select statement: %v", err)
	}

	updateState := `UPDATE lora_device SET fcnt_dn = $1, fcnt_up = $2, key_warning = $3 WHERE eui = $4`
	if ret.updateStateStatement, err = db.Prepare(updateState); err != nil {
		return nil, fmt.Errorf("unable to prepare update state statement: %v", err)
	}

	delete := `DELETE FROM lora_device WHERE eui = $1`
	if ret.deleteStatement, err = db.Prepare(delete); err != nil {
		return nil, fmt.Errorf("unable to prepare delete statement: %v", err)
	}

	update := `
		UPDATE
			lora_device
		SET
			dev_addr = $1,
			app_key = $2,
			apps_key = $3,
			nwks_key = $4,
			state = $5,
			fcnt_up = $6,
			fcnt_dn = $7,
			relaxed_counter = $8,
			key_warning = $9,
			tags = $10
		WHERE eui = $11`
	if ret.updateStatement, err = db.Prepare(update); err != nil {
		return nil, fmt.Errorf("unable to prepare device update statement: %v", err)
	}
	return &ret, nil
}

// Read nonces for device.
func (d *dbDeviceStorage) retrieveNonces(device *model.Device) error {
	rows, err := d.getNonceStatement.Query(device.DeviceEUI.String())
	if err != nil {
		return fmt.Errorf("unable to query nonces: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var nonce int
		if err := rows.Scan(&nonce); err != nil {
			logging.Warning("Unable to read DevNonce for device with EUI %s: %v", device.DeviceEUI, err)
			continue
		}
		device.DevNonceHistory = append(device.DevNonceHistory, uint16(nonce))
	}
	return nil
}

func (d *dbDeviceStorage) readDevice(row *sql.Rows) (model.Device, error) {
	ret := model.Device{}
	var devEUIStr, devAddrStr, appEUIStr, appKeyStr, appSkeyStr, nwkSkeyStr string
	var err error
	var tagBuffer []byte
	if err = row.Scan(
		&devEUIStr,
		&devAddrStr,
		&appKeyStr,
		&appSkeyStr,
		&nwkSkeyStr,
		&appEUIStr,
		&ret.State,
		&ret.FCntUp,
		&ret.FCntDn,
		&ret.RelaxedCounter,
		&ret.KeyWarning,
		&tagBuffer); err != nil {
		return ret, err
	}

	if ret.DeviceEUI, err = protocol.EUIFromString(devEUIStr); err != nil {
		return ret, fmt.Errorf("invalid Dev EUI: %v, (eui=%s)", err, devEUIStr)
	}
	if ret.DevAddr, err = protocol.DevAddrFromString(devAddrStr); err != nil {
		return ret, fmt.Errorf("invalid DevAddr for device with EUI %s (devaddr=%s)", ret.DeviceEUI, devAddrStr)
	}
	if ret.AppEUI, err = protocol.EUIFromString(appEUIStr); err != nil {
		return ret, fmt.Errorf("invalid App EUI: %v, (eui=%s)", err, appEUIStr)
	}
	if ret.AppKey, err = protocol.AESKeyFromString(appKeyStr); err != nil {
		return ret, fmt.Errorf("invalid AppKey: %v (key=%s)", err, appKeyStr)
	}
	if ret.AppSKey, err = protocol.AESKeyFromString(appSkeyStr); err != nil {
		return ret, fmt.Errorf("invalid AppSKey: %v (key=%s)", err, appSkeyStr)
	}
	if ret.NwkSKey, err = protocol.AESKeyFromString(nwkSkeyStr); err != nil {
		return ret, fmt.Errorf("invalid NwkSKey: %v (key=%s)", err, nwkSkeyStr)
	}

	tags, err := model.NewTagsFromBuffer(tagBuffer)
	if err != nil {
		return ret, fmt.Errorf("invalid tag buffer: %v (key=%s)", err, devEUIStr)
	}
	ret.Tags = *tags
	return ret, d.retrieveNonces(&ret)
}

func (d *dbDeviceStorage) getDevice(rows *sql.Rows, err error) (model.Device, error) {
	emptyDevice := model.Device{}

	if err != nil {
		return emptyDevice, err
	}
	defer rows.Close()
	if !rows.Next() {
		return emptyDevice, storage.ErrNotFound
	}
	device, err := d.readDevice(rows)
	if err != nil {
		return emptyDevice, err
	}
	return device, d.retrieveNonces(&device)
}

func (d *dbDeviceStorage) getDeviceList(rows *sql.Rows, err error) (chan model.Device, error) {
	if err != nil {
		return nil, fmt.Errorf("unable to query device list: %v", err)
	}
	outputChan := make(chan model.Device)
	go func() {
		defer rows.Close()
		defer close(outputChan)
		for rows.Next() {
			device, err := d.readDevice(rows)
			if err != nil {
				logging.Warning("unable to read device: %v; skipping it", err)
				continue
			}
			outputChan <- device
		}
	}()
	return outputChan, nil
}

// GetByDevAddr returns the device with the matching device address
func (d *dbDeviceStorage) GetByDevAddr(devAddr protocol.DevAddr) (chan model.Device, error) {
	return d.getDeviceList(d.devAddrStatement.Query(devAddr.String()))
}

func (d *dbDeviceStorage) GetByEUI(devEUI protocol.EUI) (model.Device, error) {
	return d.getDevice(d.euiStatement.Query(devEUI.String()))
}

// GetByApplicationEUI returns all devices for the given application
func (d *dbDeviceStorage) GetByApplicationEUI(appEUI protocol.EUI) (chan model.Device, error) {
	return d.getDeviceList(d.appEUIStatement.Query(appEUI.String()))
}

// Put stores the device.
func (d *dbDeviceStorage) Put(device model.Device, appEUI protocol.EUI) error {
	return d.doSQLExec(d.putStatement, func(s *sql.Stmt) (sql.Result, error) {
		return s.Exec(device.DeviceEUI.String(),
			device.DevAddr.String(),
			device.AppKey.String(),
			device.AppSKey.String(),
			device.NwkSKey.String(),
			device.AppEUI.String(),
			uint8(device.State),
			device.FCntUp,
			device.FCntDn,
			device.RelaxedCounter,
			device.KeyWarning,
			device.Tags.TagJSON())
	})
}

func (d *dbDeviceStorage) AddDevNonce(device model.Device, nonce uint16) error {
	return d.doSQLExec(d.nonceStatement, func(s *sql.Stmt) (sql.Result, error) {
		return s.Exec(device.DeviceEUI.String(), nonce)
	})
}

func (d *dbDeviceStorage) UpdateState(device model.Device) error {
	return d.doSQLExec(d.updateStateStatement, func(s *sql.Stmt) (sql.Result, error) {
		return s.Exec(device.FCntDn, device.FCntUp, device.KeyWarning, device.DeviceEUI.String())
	})
}

func (d *dbDeviceStorage) Delete(eui protocol.EUI) error {
	return d.doSQLExec(d.deleteStatement, func(s *sql.Stmt) (sql.Result, error) {
		return s.Exec(eui.String())
	})
}

func (d *dbDeviceStorage) Update(device model.Device) error {
	return d.doSQLExec(d.updateStatement, func(s *sql.Stmt) (sql.Result, error) {
		return s.Exec(
			device.DevAddr.String(),
			device.AppKey.String(),
			device.AppSKey.String(),
			device.NwkSKey.String(),
			uint8(device.State),
			device.FCntUp,
			device.FCntDn,
			device.RelaxedCounter,
			device.KeyWarning,
			device.Tags.TagJSON(),
			device.DeviceEUI.String())
	})
}
