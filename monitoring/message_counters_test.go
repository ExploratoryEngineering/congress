package monitoring

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
	"encoding/json"
	"testing"

	"github.com/ExploratoryEngineering/congress/protocol"
)

func TestMessageCounter(t *testing.T) {
	eui := protocol.EUIFromUint64(0xbeef)
	counter := NewMessageCounter(eui)

	// Encode
	buf, err := json.Marshal(&counter)
	if err != nil {
		t.Fatalf("Couldn't encode JSON: %v", err)
	}

	data := make(map[string]interface{})
	if err := json.Unmarshal(buf, &data); err != nil {
		t.Fatalf("Couldn't unmarshal JSON: %v", err)
	}
	v, exists := data["messagesIn"]
	if !exists {
		t.Fatalf("No messagesIn property on JSON object (object is %s)", string(buf))
	}
	v, exists = data["messagesOut"]
	if !exists {
		t.Fatalf("No messagesOut property on JSON object (object is %s)", string(buf))
	}
	_, ok := v.([]interface{})
	if !ok {
		t.Fatalf("messagesOut isn't an array of values (v = %T)", v)
	}
}

func TestMessageCounterList(t *testing.T) {
	counterList := newMessageCounterList()

	eui1, _ := protocol.EUIFromString("01-02-03-04-05-06-07-08")
	eui2, _ := protocol.EUIFromString("01-02-03-04-05-06-07-aa")
	c1 := counterList.Get(eui1)
	c2 := counterList.Get(eui2)

	c1.MessagesIn.Increment()
	c1.MessagesOut.Increment()

	c2.MessagesIn.Increment()
	c2.MessagesOut.Increment()

	counterList.Remove(eui1)
	counterList.Remove(eui2)
	counterList.Remove(eui1)
	counterList.Remove(protocol.EUI{})

}

func TestDefaultGWCounterList(t *testing.T) {
	eui := protocol.EUIFromUint64(1)
	GetGatewayCounters(eui)
	RemoveGatewayCounters(eui)
}

func TestDefaultAppCounterList(t *testing.T) {
	eui := protocol.EUIFromUint64(0xb00f)
	GetAppCounters(eui)
	RemoveAppCounters(eui)
	RemoveAppCounters(protocol.EUIFromUint64(0xb000fb0))
}
