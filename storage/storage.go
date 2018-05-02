package storage

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
	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
)

// ApplicationStorage is used to store and retrieve application objects in a storage backend.
type ApplicationStorage interface {
	// Put stores a new application object in the storage. If an error occurs or
	// there's an duplicate an error is returned.
	Put(application model.Application, userID model.UserID) error

	// GetByEUI retrieves the application with the corresponding EUI. If the application
	// doesn't exist an error is returned.
	GetByEUI(eui protocol.EUI, userID model.UserID) (model.Application, error)

	// GetList returns all applications available to the specified user (ID). If
	// there's an error retrieving the applications an error is returned. The
	// channel will be closed when all of the applications are returned.
	GetList(userID model.UserID) (chan model.Application, error)

	// Delete removes the application from the backend store. If the application
	// has one or more devices defined it will fail with ErrDeleteConstraint. If
	// the application isn't found it will return ErrNotFound.
	Delete(eui protocol.EUI, userID model.UserID) error

	// Update updates name key of application
	Update(application model.Application, userID model.UserID) error

	// Close closes the storage and releases allocated resources. Once Close()
	// is called it cannot do any additional operations.
	Close()
}

// DeviceStorage is used to store and retrieve device objects in a storage backend.
type DeviceStorage interface {
	// GetByDevAddr returns the device that matches the given device address.
	GetByDevAddr(devAddr protocol.DevAddr) (chan model.Device, error)

	// GetByEUI returns the device with the matching EUI
	GetByEUI(devEUI protocol.EUI) (model.Device, error)

	// GetByApplicationEUI returns all devices within the given application
	GetByApplicationEUI(appEUI protocol.EUI) (chan model.Device, error)

	// Put stores the device in the storage backend.
	Put(device model.Device, appEUI protocol.EUI) error

	// AddDevNonce adds a new device nonce to the device's history.
	AddDevNonce(device model.Device, devNonce uint16) error

	// UpdateState updates the device state (ie frame counters and key warning flag).
	UpdateState(device model.Device) error

	// Delete removes the device from the backend store. If the device isn't found
	// it will return ErrNotFound. All of the device's data will be deleted.
	Delete(eui protocol.EUI) error

	// Update updates the fields on the device
	Update(device model.Device) error

	// Close closes the storage and releases allocated resources. Once Close()
	// is called it cannot do any additional operations.
	Close()
}

// DataStorage is used to store and retrieve device data in a storage backend.
type DataStorage interface {
	// Put stores device data for the specified device
	Put(deviceEUI protocol.EUI, data model.DeviceData) error

	// GetByDeviceEUI returns the data stored for the device
	GetByDeviceEUI(deviceEUI protocol.EUI, limit int) (chan model.DeviceData, error)

	// GetByApplicationEUI returns data stored for application
	GetByApplicationEUI(applicationEUI protocol.EUI, limit int) (chan model.DeviceData, error)

	// Close closes the storage and releases allocated resources. Once Close()
	// is called it cannot do any additional operations.
	Close()

	// PutDownstream stores a new downstream message for given device. ErrConflict is returned
	// if there's already downstream message for that device
	PutDownstream(deviceEUI protocol.EUI, message model.DownstreamMessage) error

	// DeleteDownstream removes (ie cancels) the downstream message for the specified device
	DeleteDownstream(deviceEUI protocol.EUI) error

	// GetDownstream retrieves a downstream message. ErrNotFound is returned
	// if there's no downstream message for that device
	GetDownstream(deviceEUI protocol.EUI) (model.DownstreamMessage, error)

	// Update time stamps on downstream message. ErrNotFound is returned if there's no
	// downstream message for that device.
	UpdateDownstream(deviceEUI protocol.EUI, sentTime int64, ackTime int64) error
}

// GatewayStorage is used to store and retrieve gateways
type GatewayStorage interface {
	// Put stores a new gateway.
	Put(gateway model.Gateway, userID model.UserID) error

	// Remove removes a gateway from the backend storage. If the gateway isn't
	// found it will return ErrNotFound.
	Delete(eui protocol.EUI, userID model.UserID) error

	// GetList returns a list of all the available gateways.
	GetList(userID model.UserID) (chan model.Gateway, error)

	// ListAll lists all available gateways
	ListAll() (chan model.PublicGatewayInfo, error)

	// Get returns the gateway with the specified EUI.
	Get(eui protocol.EUI, userID model.UserID) (model.Gateway, error)

	// Update updates fields (and tags) on the gateway
	Update(gateway model.Gateway, userID model.UserID) error

	// Close closes the storage and releases allocated resources. Once Close()
	// is called it cannot do any additional operations.
	Close()
}

// TokenStorage is the storage layer for tokens. Tokens are primarily identified
// by the token itself. The user ID parameter is for additional sanity checks.
type TokenStorage interface {
	// Put stores a new token in the storage backend.
	Put(token model.APIToken, userID model.UserID) error

	// Delete removes a token from the storage backend.
	Delete(token string, userID model.UserID) error

	// GetList returns the complete list of tokens that the user have created.
	GetList(userID model.UserID) (chan model.APIToken, error)

	// Get returns a single token.
	Get(token string) (model.APIToken, error)

	// Update token with new tags and resource.
	Update(token model.APIToken, userID model.UserID) error

	// Close closes the storage and releases any allocated resources. Once
	// Close() is called no more operations can be performed.
	Close()
}

// Storage holds all of the storage objects
type Storage struct {
	Application    ApplicationStorage
	Device         DeviceStorage
	DeviceData     DataStorage
	Sequence       KeySequenceStorage
	Gateway        GatewayStorage
	Token          TokenStorage
	UserManagement UserManagement
	AppOutput      AppOutputStorage
}

// Close closes all of the storage instances.
func (s *Storage) Close() {
	if s.Application != nil {
		s.Application.Close()
	}
	if s.Device != nil {
		s.Device.Close()
	}
	if s.DeviceData != nil {
		s.DeviceData.Close()
	}
	if s.Sequence != nil {
		s.Sequence.Close()
	}
	if s.Gateway != nil {
		s.Gateway.Close()
	}
	if s.Token != nil {
		s.Token.Close()
	}
	if s.UserManagement != nil {
		s.UserManagement.Close()
	}
}

// KeySequenceStorage is a storage interface for key sequences. Sequences of
// keys are allocated in blocks and cannot be reused. The Increment* methods
// are atomic. The order of allocations are not important as long as they are
// unique.
type KeySequenceStorage interface {
	// AllocateKeys allocates a block of keys. Individual sequences have
	// different names. If the sequence doesn't exist it will be created
	// and set to 0. The keys will be sent on the channel until the
	// keys are exhausted and the channel will be cloesd.
	AllocateKeys(name string, interval uint64, initial uint64) (chan uint64, error)

	// Close closes the storage and releases allocated resources. Once Close()
	// is called it cannot do any additional operations.
	Close()
}

// KeyGeneratorFunc is a function that generates identifiers
type KeyGeneratorFunc func(string) uint64

// UserManagement is used to manage and set users and owners of entities.
type UserManagement interface {
	// AddOrUpdateUser adds or updates (if required) the user in the backend
	// storage. If the user doesn't exist a new owner entry will be added.
	// The key generator function is used to create a new owner entry (if
	// required). New owner entries are only created when the user is created.
	// There is no check if the owner exists.
	AddOrUpdateUser(user model.User, keyGen KeyGeneratorFunc) error

	// GetOwner will return an owner ID that can be used when asssigning
	// owners. If the owner already exists it will be returned. If not a new
	// entry in the owner table will be created.
	GetOwner(userID model.UserID) (uint64, error)

	// Close closes the storage and releases any allocated resources. Once
	// Close() is called no more operations can be performed.
	Close()
}

// AppOutputStorage is storage for the output configurations. Put, Update and
// Delete operate on single outputs while the ListAll method returns *all*
// outputs
type AppOutputStorage interface {
	// Put stores a new output
	Put(newAppOutput model.AppOutput) error

	// Update updates the configuration for the
	Update(output model.AppOutput) error

	// Delete removes the output. Outputs are also removed automatically when
	// an application is removed
	Delete(output model.AppOutput) error

	// GetByApplication returns outputs for a single application
	GetByApplication(appEUI protocol.EUI) (<-chan model.AppOutput, error)

	// List lists all of the application outputs
	ListAll() (<-chan model.AppOutput, error)
}
