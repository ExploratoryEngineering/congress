package cmac

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
	"testing"
)

/*
  Test data from RFC-4493
  --------------------------------------------------
   Subkey Generation
   K              2b7e1516 28aed2a6 abf71588 09cf4f3c
   AES-128(key,0) 7df76b0c 1ab899b3 3e42f047 b91b546f
   K1             fbeed618 35713366 7c85e08f 7236a8de
   K2             f7ddac30 6ae266cc f90bc11e e46d513b
   --------------------------------------------------

   --------------------------------------------------
   Example 1: len = 0
   M              <empty string>
   AES-CMAC       bb1d6929 e9593728 7fa37d12 9b756746
   --------------------------------------------------

   Example 2: len = 16
   M              6bc1bee2 2e409f96 e93d7e11 7393172a
   AES-CMAC       070a16b4 6b4d4144 f79bdd9d d04a287c
   --------------------------------------------------

   Example 3: len = 40
   M              6bc1bee2 2e409f96 e93d7e11 7393172a
                  ae2d8a57 1e03ac9c 9eb76fac 45af8e51
                  30c81c46 a35ce411
   AES-CMAC       dfa66747 de9ae630 30ca3261 1497c827
   --------------------------------------------------

   Example 4: len = 64
   M              6bc1bee2 2e409f96 e93d7e11 7393172a
                  ae2d8a57 1e03ac9c 9eb76fac 45af8e51
                  30c81c46 a35ce411 e5fbc119 1a0a52ef
                  f69f2445 df4f9b17 ad2b417b e66c3710
   AES-CMAC       51f0bebf 7e3b9d92 fc497417 79363cfe
   --------------------------------------------------
*/

func TestSubkey(t *testing.T) {
	key, _ := hex.DecodeString("2b7e151628aed2a6abf7158809cf4f3c")
	const k1 = "fbeed618357133667c85e08f7236a8de"
	const k2 = "f7ddac306ae266ccf90bc11ee46d513b"

	ciph, err := aes.NewCipher(key)
	if err != nil {
		t.Errorf("Error from NewCipher: %s", err)
	}
	key1, key2 := generateSubkeys(ciph)

	result1 := hex.EncodeToString(key1)
	result2 := hex.EncodeToString(key2)

	if k1 != result1 {
		t.Errorf("K1 does not match: %s != %s", k1, result1)
	}
	if k2 != result2 {
		t.Errorf("K2 does not match: %s != %s", k2, result2)
	}
}

func TestCMAC(t *testing.T) {
	testCmac := func(k string, m string, expected string) {
		key, _ := hex.DecodeString(strings.Replace(k, " ", "", -1))
		msg, _ := hex.DecodeString(strings.Replace(m, " ", "", -1))
		expectedCmac := strings.Replace(expected, " ", "", -1)
		aesCmac, err := AESCMAC(key, msg)
		if err != nil {
			t.Errorf("Did not expect error here: %s", err)
		}
		cmac := hex.EncodeToString(aesCmac)
		if cmac != expectedCmac {
			t.Errorf("Expected CMAC %s, got %s", cmac, expectedCmac)
		}
	}

	testCmac("2b7e1516 28aed2a6 abf71588 09cf4f3c",
		"",
		"bb1d6929 e9593728 7fa37d12 9b756746")

	testCmac("2b7e1516 28aed2a6 abf71588 09cf4f3c",
		"6bc1bee2 2e409f96 e93d7e11 7393172a",
		"070a16b4 6b4d4144 f79bdd9d d04a287c")

	testCmac("2b7e1516 28aed2a6 abf71588 09cf4f3c",
		"6bc1bee2 2e409f96 e93d7e11 7393172a ae2d8a57 1e03ac9c 9eb76fac 45af8e51 30c81c46 a35ce411",
		"dfa66747 de9ae630 30ca3261 1497c827")
}

func testVector(keyStr string, msgStr string, cmacStr string, t *testing.T) {
	key, err := hex.DecodeString(keyStr)
	msg, err := hex.DecodeString(msgStr)
	cmac, err := hex.DecodeString(cmacStr)

	t.Logf("Testing key=%s, msg=%s, output=%s\n", keyStr, msgStr, cmacStr)
	c, err := AESCMAC(key, msg)
	if err != nil {
		t.Fatalf("Did not expect error: %s", err)
	}
	t.Logf("Got     key=%s, msg=%s,   cmac=%s\n", keyStr, msgStr, hex.EncodeToString(c))
	for i := range cmac {
		if cmac[i] != c[i] {
			t.Fatalf("Byte at index %d (0x%02x) does not match expected output (0x%02x)", i, c[i], cmac[i])
			return
		}
	}
}

// Do benchmarking; random key, 129 byte message.
func BenchmarkCMAC(b *testing.B) {
	key := make([]byte, 16)
	msg := make([]byte, 254)
	rand.Read(key)
	rand.Read(msg)
	for i := 0; i < b.N; i++ {
		cmac, err := AESCMAC(key, msg)
		if err != nil {
			b.Errorf("Got error generating CMAC: %s", err)
		}
		if len(cmac) != constBSize {
			b.Errorf("Length of CMAC was %d bytes, expected %d bytes", len(cmac), constBSize)
		}
	}
}
