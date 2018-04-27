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

	"encoding/base64"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

// dbDataStorage is a PostgreSQL-backend data storage
type dbDataStorage struct {
	dbStore
	putStatement     *sql.Stmt
	listStatement    *sql.Stmt
	appDataList      *sql.Stmt
	putDownstream    *sql.Stmt
	deleteDownstream *sql.Stmt
	updateDownstream *sql.Stmt
	getDownstream    *sql.Stmt
}

// Close closes the resources opened by the DBDataStorage instance
func (d *dbDataStorage) Close() {
	d.putStatement.Close()
	d.listStatement.Close()
	d.appDataList.Close()
	d.putDownstream.Close()
	d.deleteDownstream.Close()
	d.updateDownstream.Close()
	d.getDownstream.Close()
}

// NewDBDataStorage creates a new DataStorage instance.
func NewDBDataStorage(db *sql.DB, userManagement storage.UserManagement) (storage.DataStorage, error) {
	ret := dbDataStorage{dbStore{db: db, userManagement: userManagement}, nil, nil, nil, nil, nil, nil, nil}
	var err error

	sqlInsert := `
		INSERT INTO
			lora_device_data (
				device_eui,
				data,
				time_stamp,
				gateway_eui,
				rssi,
				snr,
				frequency,
				data_rate,
				dev_addr)
		VALUES ($1,	$2,	$3, $4, $5, $6, $7, $8, $9)`
	if ret.putStatement, err = db.Prepare(sqlInsert); err != nil {
		return nil, fmt.Errorf("unable to prepare insert statement: %v", err)
	}

	sqlSelect := `
		SELECT
			device_eui,
			data,
			time_stamp,
			gateway_eui,
			rssi,
			snr,
			frequency,
			data_rate,
			dev_addr
		FROM
			lora_device_data
		WHERE
			device_eui = $1
		ORDER BY
			time_stamp DESC
		LIMIT $2`
	if ret.listStatement, err = db.Prepare(sqlSelect); err != nil {
		return nil, fmt.Errorf("unable to prepare list statement: %v", err)
	}

	sqlDataList := `
		SELECT d.device_eui, d.data, d.time_stamp, gateway_eui, rssi, snr, frequency, data_rate, d.dev_addr
		FROM lora_device_data d
			INNER JOIN lora_device dev ON d.device_eui = dev.eui
			INNER JOIN lora_application app ON dev.application_eui = app.eui
		WHERE app.eui = $1
			ORDER BY d.time_stamp DESC
		LIMIT $2`
	if ret.appDataList, err = db.Prepare(sqlDataList); err != nil {
		return nil, fmt.Errorf("unable to prepare app list statement: %v", err)
	}

	sqlPutDownstream := `
		INSERT INTO lora_downstream_message (
			device_eui,
			data,
			port,
			ack,
			created_time,
			sent_time,
			ack_time)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7)
	`
	if ret.putDownstream, err = db.Prepare(sqlPutDownstream); err != nil {
		return nil, fmt.Errorf("unable to prepare downstream put statement: %v", err)
	}

	sqlDeleteDownsteram := `
		DELETE FROM
			lora_downstream_message
		WHERE
			device_eui = $1
	`
	if ret.deleteDownstream, err = db.Prepare(sqlDeleteDownsteram); err != nil {
		return nil, fmt.Errorf("unable to prepare downstream delete statement: %v", err)
	}

	sqlUpdateDownstream := `
		UPDATE lora_downstream_message
			SET
				sent_time = $1,
				ack_time = $2
			WHERE
				device_eui = $3
	`
	if ret.updateDownstream, err = db.Prepare(sqlUpdateDownstream); err != nil {
		return nil, fmt.Errorf("unable to prepare downstream update statement")
	}

	sqlGetDownstream := `
		SELECT
			data,
			port,
			ack,
			created_time,
			sent_time,
			ack_time
		FROM
			lora_downstream_message
		WHERE
			device_eui = $1
	`
	if ret.getDownstream, err = db.Prepare(sqlGetDownstream); err != nil {
		return nil, fmt.Errorf("unable to prepare downstream select statement")
	}
	return &ret, nil
}

// Put stores a new data element in the backend. The element is associated with the specified DevAddr
func (d *dbDataStorage) Put(deviceEUI protocol.EUI, data model.DeviceData) error {
	return d.doSQLExec(d.putStatement, func(s *sql.Stmt) (sql.Result, error) {
		b64str := base64.StdEncoding.EncodeToString(data.Data)
		return s.Exec(deviceEUI.String(),
			b64str,
			data.Timestamp,
			data.GatewayEUI.String(),
			data.RSSI,
			data.SNR,
			data.Frequency,
			data.DataRate,
			data.DevAddr.String())
	})
}

// Decode a single row into a DeviceData instance.
func (d *dbDataStorage) readData(rows *sql.Rows) (model.DeviceData, error) {
	ret := model.DeviceData{}
	var err error
	var devEUI, dataStr, gwEUI, devAddr string
	if err = rows.Scan(&devEUI, &dataStr, &ret.Timestamp, &gwEUI, &ret.RSSI, &ret.SNR, &ret.Frequency, &ret.DataRate, &devAddr); err != nil {
		return ret, err
	}
	if ret.DeviceEUI, err = protocol.EUIFromString(devEUI); err != nil {
		return ret, err
	}
	if ret.Data, err = base64.StdEncoding.DecodeString(dataStr); err != nil {
		return ret, err
	}
	if ret.GatewayEUI, err = protocol.EUIFromString(gwEUI); err != nil {
		return ret, err
	}
	if ret.DevAddr, err = protocol.DevAddrFromString(devAddr); err != nil {
		return ret, err
	}
	return ret, nil
}

func (d *dbDataStorage) doQuery(stmt *sql.Stmt, eui string, limit int) (chan model.DeviceData, error) {
	rows, err := stmt.Query(eui, limit)
	if err != nil {
		return nil, fmt.Errorf("unable to query device data for device with EUI %s: %v", eui, err)
	}
	outputChan := make(chan model.DeviceData)
	go func() {
		defer rows.Close()
		defer close(outputChan)
		for rows.Next() {
			ret, err := d.readData(rows)
			if err != nil {
				logging.Warning("Unable to decode data for device with EUI %s: %v", eui, err)
				continue
			}
			outputChan <- ret
		}
	}()
	return outputChan, nil
}

// GetByDeviceEUI retrieves all of the data stored for that DevAddr
func (d *dbDataStorage) GetByDeviceEUI(deviceEUI protocol.EUI, limit int) (chan model.DeviceData, error) {
	return d.doQuery(d.listStatement, deviceEUI.String(), limit)
}

func (d *dbDataStorage) GetByApplicationEUI(applicationEUI protocol.EUI, limit int) (chan model.DeviceData, error) {
	return d.doQuery(d.appDataList, applicationEUI.String(), limit)
}

func (d *dbDataStorage) PutDownstream(deviceEUI protocol.EUI, message model.DownstreamMessage) error {
	return d.doSQLExec(d.putDownstream, func(s *sql.Stmt) (sql.Result, error) {
		return s.Exec(
			deviceEUI.String(),
			message.Data,
			message.Port,
			message.Ack,
			message.CreatedTime,
			message.SentTime,
			message.AckTime)
	})
}

func (d *dbDataStorage) DeleteDownstream(deviceEUI protocol.EUI) error {
	return d.doSQLExec(d.deleteDownstream, func(s *sql.Stmt) (sql.Result, error) {
		return s.Exec(deviceEUI.String())
	})
}

func (d *dbDataStorage) GetDownstream(deviceEUI protocol.EUI) (model.DownstreamMessage, error) {
	ret := model.NewDownstreamMessage(deviceEUI, 0)

	rows, err := d.getDownstream.Query(deviceEUI.String())
	if err != nil {
		return ret, fmt.Errorf("unable to query for downstream message: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return ret, storage.ErrNotFound
	}
	if err := rows.Scan(&ret.Data, &ret.Port, &ret.Ack, &ret.CreatedTime, &ret.SentTime, &ret.AckTime); err != nil {
		return ret, fmt.Errorf("unable to read fields from downstream result: %v", err)
	}
	return ret, nil
}

func (d *dbDataStorage) UpdateDownstream(deviceEUI protocol.EUI, sentTime int64, ackTime int64) error {
	return d.doSQLExec(d.updateDownstream, func(s *sql.Stmt) (sql.Result, error) {
		return s.Exec(
			sentTime,
			ackTime,
			deviceEUI.String())
	})
}
