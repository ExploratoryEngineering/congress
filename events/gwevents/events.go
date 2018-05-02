package gwevents

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
type gwEventType string

// GwEvent types are OOB events for the gateway. They will be sent
// as a debugging aid for gateways. The gateway interface(s) forwards all
// gateway events to a buffered channel which will distribute the events
// to listeners.
type GwEvent struct {
	Type gwEventType `json:"event"`          // EventType holds the event type (see constants)
	Data string      `json:"data,omitempty"` // The data sent or received from the gateway (if applicable)
}

// NewInactive creates a new inactive event
func NewInactive() GwEvent {
	return GwEvent{gwEventType("Inactive"), ""}
}

// NewKeepAlive creates a new keepalive event
func NewKeepAlive() GwEvent {
	return GwEvent{gwEventType("KeepAlive"), ""}
}

// NewRx creates a new Rx event for the gateway
func NewRx(data string) GwEvent {
	return GwEvent{gwEventType("Rx"), data}
}

// NewTx creates a new Tx event for the gateway
func NewTx(data string) GwEvent {
	return GwEvent{gwEventType("Tx"), data}
}
