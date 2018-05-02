package protocol

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
import "errors"

var (
	// ErrBufferTruncated is returned when the buffer is too short to encode or decode
	ErrBufferTruncated = errors.New("buffer too short")
	// ErrNilError is returned when one or more parameter is nil
	ErrNilError = errors.New("parameter is nil")
	// ErrParameterOutOfRange is returned when one of the parameters are out of range
	ErrParameterOutOfRange = errors.New("parameter out of range")
	// ErrInvalidParameterFormat is returned when a supplied parameter is invalid
	ErrInvalidParameterFormat = errors.New("invalid parameter format")
	// ErrCryptoError is returned when there's an error with the crypto library. It's not a common occurrence.
	ErrCryptoError = errors.New("crypto system error")
	// ErrInvalidSource is returned when the input buffer contains corrupted data
	ErrInvalidSource = errors.New("source buffer is corrupted")
	// ErrInvalidMessageType is returned when the message type isn't supported
	ErrInvalidMessageType = errors.New("invalid message type")
	// ErrInvalidLoRaWANVersion is returned when the LoRaWAN version is unsupported
	ErrInvalidLoRaWANVersion = errors.New("unsupported LoRaWAN version")
	// This is used internally to signal unknown MAC command
	errUnknownMAC = errors.New("unknown MAC command")
)
