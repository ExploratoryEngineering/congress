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
import "testing"

func TestInvalidStrings(t *testing.T) {
	// Empty string
	_, err := AESKeyFromString("")
	if err == nil {
		t.Error("Expected error when using empty string")
	}

	// Invalid hex string
	_, err = AESKeyFromString("foo bar _ baz")
	if err == nil {
		t.Error("Expected error when using invalid string")
	}

	// Too short byte buffer
	_, err = AESKeyFromString("0102030405060708")
	if err == nil {
		t.Error("Expected error when using short buffer")
	}

	// Too long byte buffer
	_, err = AESKeyFromString("010203040506070801020304050607080102030405060708")
	if err == nil {
		t.Error("Expected error when using short buffer")
	}
}

func TestStringConversion(t *testing.T) {
	key, err := AESKeyFromString("01020304050607080102030405060708")
	if err != nil {
		t.Error("Got error converting bytes: ", err)
	}

	expected := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}
	for i := range key.Key {
		if key.Key[i] != expected[i] {
			t.Error("Incorrect byte at pos ", i, ": ", key.Key[i])
		}
	}
}

func TestToString(t *testing.T) {
	keybytes := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}
	key := AESKey{Key: keybytes}

	if key.String() != "01020304050607080102030405060708" {
		t.Error("Not expected output")
	}
}

func TestNwkSKeyFromNonces(t *testing.T) {
	appNonce := [3]byte{01, 02, 03}
	netID := uint32(0x00AABBCC)
	devNonce := uint16(0xabcd)
	appKey, _ := AESKeyFromString("0102-0304-0506-0708-0102-0304-0506-0708")
	// Just make a key and make sure it isn't the same as the original
	key, err := NwkSKeyFromNonces(appKey, appNonce, netID, devNonce)
	if err != nil {
		t.Fatal("Got error generating key: ", err)
	}
	if key == appKey {
		t.Fatal("Key isn't any different")
	}
}

func TestAppSKeyFromNonces(t *testing.T) {
	appNonce := [3]byte{01, 02, 03}
	netID := uint32(0x00AABBCC)
	devNonce := uint16(0xabcd)
	appKey, _ := AESKeyFromString("0102-0304-0506-0708-0102-0304-0506-0708")
	// Just make a key and make sure it isn't the same as the original
	key, err := AppSKeyFromNonces(appKey, appNonce, netID, devNonce)
	if err != nil {
		t.Fatal("Got error generating key: ", err)
	}
	if key == appKey {
		t.Fatal("Key isn't any different")
	}
}

func TestNewAESKey(t *testing.T) {
	if _, err := NewAESKey(); err != nil {
		t.Fatal("Got error creating key. Shouldn't get that.")
	}
}
