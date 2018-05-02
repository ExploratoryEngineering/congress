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
	"sync"

	"github.com/ExploratoryEngineering/congress/protocol"
)

// MessageCounter holds the counters for a single gateway or application
type MessageCounter struct {
	MessagesIn  *TimeSeries `json:"messagesIn"`
	MessagesOut *TimeSeries `json:"messagesOut"`
}

// NewMessageCounter creates a new GatewayCounter instance
func NewMessageCounter(eui protocol.EUI) *MessageCounter {
	return &MessageCounter{
		MessagesIn:  NewTimeSeries(Minutes),
		MessagesOut: NewTimeSeries(Minutes),
	}
}

// Internal type to keep track of gateway counters
type messageCounterList struct {
	counters map[protocol.EUI]*MessageCounter
	mutex    *sync.Mutex
}

// Get returns a gateway counter.
func (g *messageCounterList) Get(eui protocol.EUI) *MessageCounter {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	ret, exists := g.counters[eui]
	if !exists {
		ret = NewMessageCounter(eui)
		g.counters[eui] = ret
	}
	return ret
}

func (g *messageCounterList) Remove(eui protocol.EUI) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	delete(g.counters, eui)
}

// Create a new list of gateway counters
func newMessageCounterList() messageCounterList {
	return messageCounterList{make(map[protocol.EUI]*MessageCounter), &sync.Mutex{}}
}

var gwCounters = newMessageCounterList()

// GetGatewayCounters returns the gateway counters for the specified EUI. If the
// counters doesn't exist, a new set of counters will be created.
func GetGatewayCounters(eui protocol.EUI) *MessageCounter {
	return gwCounters.Get(eui)
}

// RemoveGatewayCounters removes the associated gateway counters
func RemoveGatewayCounters(eui protocol.EUI) {
	gwCounters.Remove(eui)
}

var appCounters = newMessageCounterList()

// GetAppCounters returns the gateway counters for the specified EUI. If the
// counters doesn't exist, a new set of counters will be created.
func GetAppCounters(eui protocol.EUI) *MessageCounter {
	return gwCounters.Get(eui)
}

// RemoveAppCounters removes the associated gateway counters
func RemoveAppCounters(eui protocol.EUI) {
	gwCounters.Remove(eui)
}
