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
import (
	"crypto/aes"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// AESKey represents an AES-128 key
type AESKey struct {
	Key [16]byte
}

// AESKeyFromString converts a byte string (with optional spaces) into an AES key
func AESKeyFromString(keyStr string) (AESKey, error) {
	tmpBuf, err := hex.DecodeString(strings.Replace(keyStr, " ", "", -1))
	if err != nil {
		return AESKey{}, err
	}
	if len(tmpBuf) != 16 {
		return AESKey{}, ErrInvalidParameterFormat
	}
	ret := AESKey{}
	copy(ret.Key[:], tmpBuf)
	return ret, nil
}

func (k AESKey) String() string {
	return hex.EncodeToString(k.Key[:])
}

// NwkSKeyFromNonces generates a new network session key derived from the application key. [6.2.5]
func NwkSKeyFromNonces(appKey AESKey, appNonce [3]byte, netID uint32, devNonce uint16) (AESKey, error) {
	return keyFromNonce(appKey, 1, appNonce, netID, devNonce)
}

// AppSKeyFromNonces generates a new application session key derived from the application key. [6.2.5]
func AppSKeyFromNonces(appKey AESKey, appNonce [3]byte, netID uint32, devNonce uint16) (AESKey, error) {
	return keyFromNonce(appKey, 2, appNonce, netID, devNonce)
}

func keyFromNonce(appKey AESKey, prefix byte, appNonce [3]byte, netID uint32, devNonce uint16) (AESKey, error) {
	buffer := make([]byte, 16)
	buffer[0] = prefix
	buffer[1] = appNonce[0]
	buffer[2] = appNonce[1]
	buffer[3] = appNonce[2]
	buffer[4] = byte((netID >> 16) & 0xFF)
	buffer[5] = byte((netID >> 8) & 0xFF)
	buffer[6] = byte((netID >> 0) & 0xFF)
	buffer[7] = byte((devNonce >> 8) & 0xFF)
	buffer[8] = byte((devNonce >> 0) & 0xFF)

	aesCipher, err := aes.NewCipher(appKey.Key[:])
	if err != nil {
		return AESKey{}, err
	}

	ret := AESKey{}
	aesCipher.Encrypt(ret.Key[:], buffer)
	return ret, nil
}

// NewAESKey creates a new AES key from the secure random generator
func NewAESKey() (AESKey, error) {
	ret := AESKey{}
	n, err := rand.Read(ret.Key[:])
	if err != nil {
		return AESKey{}, err
	}
	if n != len(ret.Key) {
		return AESKey{}, ErrCryptoError
	}
	return ret, nil
}
