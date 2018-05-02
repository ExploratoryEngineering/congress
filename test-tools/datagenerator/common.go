package main

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
	"math/rand"
	"net"

	"github.com/ExploratoryEngineering/congress/protocol"
)

func randomEUI() protocol.EUI {
	var buf [8]byte
	rand.Read(buf[:])
	return protocol.EUI{Octets: buf}
}

func randomIP() net.IP {
	var buf [4]byte
	rand.Read(buf[:])
	return net.IP(buf[:])
}

func randomAesKey() protocol.AESKey {
	var buf [16]byte
	rand.Read(buf[:])
	return protocol.AESKey{Key: buf}
}

func randomDevAddr() protocol.DevAddr {
	return protocol.DevAddrFromUint32(rand.Uint32())
}
