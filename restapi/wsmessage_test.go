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
import "testing"
import "encoding/json"

func marshalUnmarshalMsg(t *testing.T, msg wsMessage) wsMessage {
	bytes, err := json.Marshal(&msg)
	if err != nil {
		t.Fatalf("Got error marshaling message: %v", err)
	}
	var ret wsMessage
	if err := json.Unmarshal(bytes, &ret); err != nil {
		t.Fatalf("Got error unmarshaling message: %v", err)
	}
	return ret
}

func TestCreateMessages(t *testing.T) {
	keepAliveMsg := marshalUnmarshalMsg(t, newWSKeepAlive())
	if keepAliveMsg.Type != "KeepAlive" {
		t.Fatal("Expected message type to be KeepAlive")
	}

	errorMsg := marshalUnmarshalMsg(t, newWSError("This is an error"))
	if errorMsg.Message != "This is an error" {
		t.Fatal("Expected message to be the error message")
	}
	dataMsg := marshalUnmarshalMsg(t, newWSData(&apiDeviceData{Data: "Hello hello"}))
	if dataMsg.Data.Data != "Hello hello" {
		t.Fatal("Data message isn't kept")
	}
}
