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
	"fmt"

	"database/sql"

	"net"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

type dbGatewayStorage struct {
	dbStore
	putStatement        *sql.Stmt // Prepare statement for put operation
	deleteStatement     *sql.Stmt // Prepare statement for delete operation
	listStatement       *sql.Stmt // Prepare statement for select
	getStatement        *sql.Stmt // Prepare statement for select
	getSysStatement     *sql.Stmt // Prepare statement for system get (ie all gateways)
	updateStatement     *sql.Stmt // Prepare statement for gatway update
	publicListStatement *sql.Stmt
}

func (d *dbGatewayStorage) Close() {
	d.putStatement.Close()
	d.deleteStatement.Close()
	d.listStatement.Close()
	d.getStatement.Close()
	d.getSysStatement.Close()
	d.updateStatement.Close()
	d.publicListStatement.Close()
}

// NewDBGatewayStorage returns a DB-backed GatewayStorage implementation.
func NewDBGatewayStorage(db *sql.DB, userManagement storage.UserManagement) (storage.GatewayStorage, error) {
	ret := dbGatewayStorage{dbStore{db: db, userManagement: userManagement}, nil, nil, nil, nil, nil, nil, nil}

	var err error
	sqlSelect := `
		SELECT
			gw.gateway_eui,
			gw.latitude,
			gw.longitude,
			gw.altitude,
			gw.ip,
			gw.strict_ip,
			gw.tags
		FROM
			lora_gateway gw,
			lora_owner o
		WHERE
			gw.owner_id = o.owner_id AND o.user_id = $1`

	if ret.listStatement, err = db.Prepare(sqlSelect); err != nil {
		return nil, fmt.Errorf("unable to prepare select statement: %v", err)
	}

	sqlInsert := `
		INSERT INTO lora_gateway (
			gateway_eui,
			latitude,
			longitude,
			altitude,
			ip,
			strict_ip,
			owner_id,
			tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	if ret.putStatement, err = db.Prepare(sqlInsert); err != nil {
		return nil, fmt.Errorf("unable to prepare insert statement: %v", err)
	}

	sqlDelete := `
		DELETE FROM
			lora_gateway gw
		USING
			lora_owner o
		WHERE
			gw.gateway_eui = $1 AND gw.owner_id = o.owner_id AND o.user_id = $2`
	if ret.deleteStatement, err = db.Prepare(sqlDelete); err != nil {
		return nil, fmt.Errorf("unable to prepare delete statement: %v", err)
	}

	sqlSelectOne := `
		SELECT
			gw.gateway_eui,
			gw.latitude,
			gw.longitude,
			gw.altitude,
			gw.ip,
			gw.strict_ip,
			gw.tags
		FROM
			lora_gateway gw,
			lora_owner o
		WHERE
			gw.gateway_eui = $1 AND gw.owner_id = o.owner_id AND o.user_id = $2`
	if ret.getStatement, err = db.Prepare(sqlSelectOne); err != nil {
		return nil, fmt.Errorf("unable to prepare select statement: %v", err)
	}

	sysGetStatement := `
		SELECT
			gw.gateway_eui,
			gw.latitude,
			gw.longitude,
			gw.altitude,
			gw.ip,
			gw.strict_ip,
			gw.tags
		FROM
			lora_gateway gw
		WHERE
			gw.gateway_eui = $1`
	if ret.getSysStatement, err = db.Prepare(sysGetStatement); err != nil {
		return nil, fmt.Errorf("unable to prepare system get statement: %v", err)
	}

	updateStatement := `
		UPDATE
			lora_gateway gw
		SET
			latitude = $1, longitude = $2, altitude = $3, ip = $4, strict_ip = $5, tags = $6
		FROM
			lora_owner o
		WHERE
			gw.gateway_eui = $7 AND gw.owner_id = o.owner_id AND o.user_id = $8
	`
	if ret.updateStatement, err = db.Prepare(updateStatement); err != nil {
		return nil, fmt.Errorf("unable to prepare update statement: %v", err)
	}

	publicListStatement := `SELECT gateway_eui, latitude, longitude, altitude FROM lora_gateway`
	if ret.publicListStatement, err = db.Prepare(publicListStatement); err != nil {
		return nil, fmt.Errorf("unable to prepare public gateway statement: %v", err)
	}
	return &ret, nil
}

func (d *dbGatewayStorage) readGateway(rows *sql.Rows) (model.Gateway, error) {
	var euiStr, ipStr string
	var err error
	var json []uint8
	gw := model.NewGateway()
	if err := rows.Scan(&euiStr, &gw.Latitude, &gw.Longitude, &gw.Altitude, &ipStr, &gw.StrictIP, &json); err != nil {
		return gw, err
	}
	if gw.GatewayEUI, err = protocol.EUIFromString(euiStr); err != nil {
		return gw, err
	}
	gw.IP = net.ParseIP(ipStr)
	tags, err := model.NewTagsFromBuffer(json[:])
	if err != nil {
		return gw, err
	}
	gw.Tags = *tags
	return gw, nil
}

func (d *dbGatewayStorage) getGwList(rows *sql.Rows, err error) (chan model.Gateway, error) {
	if err != nil {
		return nil, err
	}
	ret := make(chan model.Gateway)
	go func() {
		defer rows.Close()
		defer close(ret)
		for rows.Next() {
			gw, err := d.readGateway(rows)
			if err != nil {
				logging.Warning("Unable to read gateway list: %v", err)
				continue
			}
			ret <- gw
		}
	}()
	return ret, nil
}
func (d *dbGatewayStorage) GetList(userID model.UserID) (chan model.Gateway, error) {
	return d.getGwList(d.listStatement.Query(string(userID)))
}

func (d *dbGatewayStorage) ListAll() (chan model.PublicGatewayInfo, error) {
	rows, err := d.publicListStatement.Query()
	if err != nil {
		return nil, fmt.Errorf("unable to query public gateways: %v", err)
	}
	outputChan := make(chan model.PublicGatewayInfo)
	go func() {
		defer rows.Close()
		defer close(outputChan)
		for rows.Next() {
			gw := model.PublicGatewayInfo{}
			if err := rows.Scan(&gw.EUI, &gw.Latitude, &gw.Longitude, &gw.Altitude); err != nil {
				logging.Warning("Unable to scan public gateway: %v", err)
				continue
			}
			outputChan <- gw
		}
	}()
	return outputChan, nil
}

func (d *dbGatewayStorage) getGateway(rows *sql.Rows, err error) (model.Gateway, error) {
	if err != nil {
		return model.Gateway{}, err
	}

	defer rows.Close()

	if !rows.Next() {
		return model.Gateway{}, storage.ErrNotFound
	}

	gw, err := d.readGateway(rows)
	if err != nil {
		return model.Gateway{}, err
	}
	return gw, nil
}

func (d *dbGatewayStorage) Get(eui protocol.EUI, userID model.UserID) (model.Gateway, error) {
	if userID == model.SystemUserID {
		return d.getGateway(d.getSysStatement.Query(eui.String()))
	}
	return d.getGateway(d.getStatement.Query(eui.String(), string(userID)))
}

func (d *dbGatewayStorage) Put(gateway model.Gateway, userID model.UserID) error {
	return d.doSQLExecWithOwner(d.putStatement, func(s *sql.Stmt, ownerID uint64) (sql.Result, error) {
		return s.Exec(
			gateway.GatewayEUI.String(),
			gateway.Latitude,
			gateway.Longitude,
			gateway.Altitude,
			gateway.IP.String(),
			gateway.StrictIP,
			ownerID,
			gateway.TagJSON())
	}, userID)
}

func (d *dbGatewayStorage) Delete(eui protocol.EUI, userID model.UserID) error {
	return d.doSQLExecWithOwner(d.deleteStatement, func(s *sql.Stmt, ownerID uint64) (sql.Result, error) {
		return s.Exec(eui.String(), string(userID))
	}, userID)
}

func (d *dbGatewayStorage) Update(gateway model.Gateway, userID model.UserID) error {
	return d.doSQLExecWithOwner(d.updateStatement, func(s *sql.Stmt, ownerID uint64) (sql.Result, error) {
		return s.Exec(gateway.Latitude, gateway.Longitude, gateway.Altitude,
			gateway.IP.String(), gateway.StrictIP, gateway.Tags.TagJSON(), gateway.GatewayEUI.String(), string(userID))
	}, userID)
}
