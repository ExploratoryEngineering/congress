package frequency

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
// Stub functions for frequency management.

import (
	"github.com/ExploratoryEngineering/congress/protocol"
)

// TODO: Replace with proper frequency management. These values are the defaults
// for the EU868 band and won't work for other bands.

// GetDLSettingsOTAA returns the DLSettings value returned during OTAA
// join procedure. This returns the default values for now.
func GetDLSettingsOTAA() protocol.DLSettings {
	return protocol.DLSettings{
		RX1DRoffset: 0,
		RX2DataRate: 5,
	}
}

// GetRxDelayOTAA returns the RxDelay parameter for OTAA join procedures. This
// function always returns 1, the default value.
func GetRxDelayOTAA() uint8 {
	return 1
}

// GetCFListOTAA returns the CFList type used during OTAA. It always returns
// the default values.
func GetCFListOTAA() protocol.CFList {
	return protocol.CFList{}
}
