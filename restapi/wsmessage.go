package restapi

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
// wsMessage has a message type and a message body
type wsMessage struct {
	Type    string         `json:"type"`
	Message string         `json:"message,omitempty"`
	Data    *apiDeviceData `json:"data,omitempty"`
}

func newWSKeepAlive() wsMessage {
	return wsMessage{"KeepAlive", "", nil}
}
func newWSError(errMsg string) wsMessage {
	return wsMessage{"Error", errMsg, nil}
}
func newWSData(data *apiDeviceData) wsMessage {
	return wsMessage{"DeviceData", "", data}
}
