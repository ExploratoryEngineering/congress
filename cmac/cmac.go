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
// AES-CMAC implemented according to https://tools.ietf.org/html/rfc4493

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"math"

	"github.com/ExploratoryEngineering/logging"
)

/*
   +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
   +                   Algorithm AES-CMAC                              +
   +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
   +                                                                   +
   +   Input    : K    ( 128-bit key )                                 +
   +            : M    ( message to be authenticated )                 +
   +            : len  ( length of the message in octets )             +
   +   Output   : T    ( message authentication code )                 +
   +                                                                   +
   +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
   +   Constants: const_Zero is 0x00000000000000000000000000000000     +
   +              const_Bsize is 16                                    +
   +                                                                   +
   +   Variables: K1, K2 for 128-bit subkeys                           +
   +              M_i is the i-th block (i=1..ceil(len/const_Bsize))   +
   +              M_last is the last block xor-ed with K1 or K2        +
   +              n      for number of blocks to be processed          +
   +              r      for number of octets of last block            +
   +              flag   for denoting if last block is complete or not +
   +                                                                   +
   +   Step 1.  (K1,K2) := Generate_Subkey(K);                         +
   +   Step 2.  n := ceil(len/const_Bsize);                            +
   +   Step 3.  if n = 0                                               +
   +            then                                                   +
   +                 n := 1;                                           +
   +                 flag := false;                                    +
   +            else                                                   +
   +                 if len mod const_Bsize is 0                       +
   +                 then flag := true;                                +
   +                 else flag := false;                               +
   +                                                                   +
   +   Step 4.  if flag is true                                        +
   +            then M_last := M_n XOR K1;                             +
   +            else M_last := padding(M_n) XOR K2;                    +
   +   Step 5.  X := const_Zero;                                       +
   +   Step 6.  for i := 1 to n-1 do                                   +
   +                begin                                              +
   +                  Y := X XOR M_i;                                  +
   +                  X := AES-128(K,Y);                               +
   +                end                                                +
   +            Y := M_last XOR X;                                     +
   +            T := AES-128(K,Y);                                     +
   +   Step 7.  return T;                                              +
   +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
*/

const constBSize int = 16

var constZero []byte
var constRb []byte

func init() {
	constZero, _ = hex.DecodeString("00000000000000000000000000000000")
	constRb, _ = hex.DecodeString("00000000000000000000000000000087")
}

// AESCMAC calculates CMAC for the given key and buffer
func AESCMAC(key []byte, buffer []byte) ([]byte, error) {
	ciph, err := aes.NewCipher(key)
	if err != nil {
		logging.Error("Unable to create new AES cipher: %s", err)
	}

	// Step 1
	k1, k2 := generateSubkeys(ciph)

	// Step 2
	n := int(math.Ceil(float64(len(buffer)) / float64(constBSize)))

	// Step 3
	var flag bool
	if n == 0 {
		n = 1
		flag = false
	} else {
		flag = (len(buffer)%constBSize == 0)
	}

	// Step 4
	mLast := make([]byte, constBSize)
	if flag {
		mn := buffer[(n-1)*constBSize:]
		mLast = xor(mn, k1)
	} else {
		mn := buffer[(n-1)*constBSize:]
		mn = padblock(mn, constBSize)
		mLast = xor(mn, k2)
	}

	// Step 5
	x := append([]byte{}, constZero...)

	// Step 6
	y := make([]byte, constBSize)
	pos := 0
	for i := 1; i < n; i++ {
		mi := buffer[pos:(pos + constBSize)]
		y = xor(x, mi)
		ciph.Encrypt(x, y)
		pos += constBSize
	}

	y = xor(mLast, x)
	t := make([]byte, constBSize)
	ciph.Encrypt(t, y)

	// Step 7
	return t, nil
}

/*
   +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
   +                    Algorithm Generate_Subkey                      +
   +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
   +                                                                   +
   +   Input    : K (128-bit key)                                      +
   +   Output   : K1 (128-bit first subkey)                            +
   +              K2 (128-bit second subkey)                           +
   +-------------------------------------------------------------------+
   +                                                                   +
   +   Constants: const_Zero is 0x00000000000000000000000000000000     +
   +              const_Rb   is 0x00000000000000000000000000000087     +
   +   Variables: L          for output of AES-128 applied to 0^128    +
   +                                                                   +
   +   Step 1.  L := AES-128(K, const_Zero);                           +
   +   Step 2.  if MSB(L) is equal to 0                                +
   +            then    K1 := L << 1;                                  +
   +            else    K1 := (L << 1) XOR const_Rb;                   +
   +   Step 3.  if MSB(K1) is equal to 0                               +
   +            then    K2 := K1 << 1;                                 +
   +            else    K2 := (K1 << 1) XOR const_Rb;                  +
   +   Step 4.  return K1, K2;                                         +
   +                                                                   +
   +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
*/

func generateSubkeys(ciph cipher.Block) (k1 []byte, k2 []byte) {
	// Step 1
	l := make([]byte, constBSize)
	ciph.Encrypt(l, constZero)

	// Step 2
	msb := (l[0] >> 7)
	if msb == 0 {
		k1 = shiftLeft(l)
	} else {
		k1 = xor(shiftLeft(l), constRb)
	}
	// Step 3
	msb = (k1[0] >> 7)
	if msb == 0 {
		k2 = shiftLeft(k1)
	} else {
		k2 = xor(shiftLeft(k1), constRb)
	}
	// Step 4
	return
}
