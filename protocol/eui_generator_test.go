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

func TestMAPatterns(t *testing.T) {
	mal, err := NewMA([]byte{0x00, 0x09, 0x09})
	if err != nil {
		t.Fatal("Could not create MA-L: ", err)
	}

	mam, err := NewMA([]byte{0x01, 0x02, 0x03, 0x40})
	if err != nil {
		t.Fatal("Could not create MA-M: ", err)
	}

	mas, err := NewMA([]byte{0x70, 0xb3, 0xd5, 0x27, 0xD0})
	if err != nil {
		t.Fatal("Could not create MA-S: ", err)
	}

	sourceEUI, _ := EUIFromString("AA-BB-CC-DD-EE-FF-AA-BB")

	malEUI := mal.Combine(sourceEUI)
	expectedEUI, _ := EUIFromString("00-09-09-DD-EE-FF-AA-BB")
	if malEUI != expectedEUI {
		t.Error("MA-L EUI isn't the expected one. Expected ", expectedEUI, " got ", malEUI)
	}

	mamEUI := mam.Combine(sourceEUI)
	expectedEUI, _ = EUIFromString("01-02-03-4D-EE-FF-AA-BB")
	if mamEUI != expectedEUI {
		t.Error("MA-M EUI isn't the expected one. Expected ", expectedEUI, " got ", mamEUI)
	}

	masEUI := mas.Combine(sourceEUI)
	expectedEUI, _ = EUIFromString("70-b3-d5-27-dE-FF-AA-BB")
	if masEUI != expectedEUI {
		t.Error("MA-S EUI isn't the expected one. Expected ", expectedEUI, " got ", masEUI)
	}
}

func TestInvalidMAPatterns(t *testing.T) {
	if _, err := NewMA(nil); err == nil {
		t.Error("Did not get error with nil MA pattern")
	}

	if _, err := NewMA([]byte{}); err == nil {
		t.Error("Did not get error with empty MA pattern")
	}

	if _, err := NewMA([]byte{01}); err == nil {
		t.Error("Did not get error with too short MA pattern")
	}

	if _, err := NewMA([]byte{01, 02}); err == nil {
		t.Error("Did not get error with too short MA pattern")
	}

	if _, err := NewMA([]byte{01, 02, 0x03, 04, 05, 06}); err == nil {
		t.Error("Did not get error with too short MA pattern")
	}

}

func TestNewDeviceEUI(t *testing.T) {
	ma, _ := NewMA([]byte{0x00, 0x09, 0x09})
	netID := uint32(0x5F6F7F)
	nwkAddr := uint32(0x01ABCDEF)

	deviceEUI := NewDeviceEUI(ma, netID, nwkAddr)

	//     8.......7.......6.......5.......4.......3.......2.......1.......0
	//     |                               |NwkID-|                          7 bits
	//     |                                      |NwkAddr (counter)-------| 25 bits
	//     |              |NetID------------------|                          24 bits
	//     |MA-L-------------------|                                         24 bits
	//     |MA-M-----------------------|                                     28 bits
	//     |MA-S------------------------------|                              36 bits

	// NetID combined with nwkAddr is going to be (5F6F7F << 1) | nwkAddr = 0xBEDEFE000000|01ABCDEF = 0xBEDEFFABCDEF
	// Masked with the MA we'll end up with the string below
	expectedEUI, _ := EUIFromString("00-09-09-DE-FF-AB-CD-EF")
	if expectedEUI != deviceEUI {
		t.Fatal("Did not get the DeviceEUI I was looking for (got ", deviceEUI, " wanted ", expectedEUI)
	}
}

// Test application EUI but use MA-M range for fun and profit.
func TestNewApplicationEUI(t *testing.T) {
	ma, _ := NewMA([]byte{0x00, 0x09, 0x09, 0xA0})
	netID := uint32(0x5F6F7F)
	counter := uint32(0x01ABCDEF)

	applicationEUI := NewApplicationEUI(ma, netID, counter)
	expectedEUI, _ := EUIFromString("00-09-09-AE-FF-AB-CD-EF")
	if expectedEUI != applicationEUI {
		t.Fatal("Did not get the ApplicationEUI I was looking for (got ", applicationEUI, " wanted ", expectedEUI)
	}
}

// Network EUI is even simpler since the counter bits are all zeroed out. Use
// MA-S for more variation
func TestNewNetworkEUI(t *testing.T) {
	ma, _ := NewMA([]byte{0x00, 0x09, 0x09, 0x10, 0x20})
	netID := uint32(0x01)

	networkEUI := NewNetworkEUI(ma, netID)
	expectedEUI, _ := EUIFromString("00-09-09-10-22-00-00-00")
	if expectedEUI != networkEUI {
		t.Fatal("Did not get the NetworkEUI I was looking for (got ", networkEUI, " wanted ", expectedEUI)
	}

	// Increase netID by 1 to get an ever so slightly different EUI
	netID++
	networkEUI = NewNetworkEUI(ma, netID)
	expectedEUI, _ = EUIFromString("00-09-09-10-24-00-00-00")
	if expectedEUI != networkEUI {
		t.Fatal("Did not get the NetworkEUI I was looking for (got ", networkEUI, " wanted ", expectedEUI)
	}

}

func TestString(t *testing.T) {
	ma, _ := NewMA([]byte{0, 1, 2})
	if ma.String() != "00-01-02" {
		t.Error("Unexpected MA-L format: ", ma.String())
	}

	ma, _ = NewMA([]byte{0, 1, 2, 3})
	if ma.String() != "00-01-02-03" {
		t.Error("Unexpected MA-M format: ", ma.String())
	}

	ma, _ = NewMA([]byte{0, 1, 2, 3, 4})
	if ma.String() != "00-01-02-03-04" {
		t.Error("Unexpected MA-S format: ", ma.String())
	}
}
