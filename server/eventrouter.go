package server

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
	"github.com/ExploratoryEngineering/logging"
)

type route struct {
	eui string
	ch  chan interface{}
}

// EventRouter is a channel event router. It will route events (or entities)
// based on the EUI. There may be multiple subscribers to the same EUI and each
// will receive a separate event. The channels are buffered and if the subscribers
// can't keep up with the events they will be dropped silently by the router.
type EventRouter struct {
	routes        []route
	mutex         *sync.Mutex
	channelLength int
}

// NewEventRouter creates a new event router
func NewEventRouter(channelLength int) EventRouter {
	return EventRouter{
		make([]route, 0),
		&sync.Mutex{},
		channelLength,
	}
}

// Subscribe subscribes to events for a particular gateway
func (e *EventRouter) Subscribe(eui protocol.EUI) <-chan interface{} {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	events := make(chan interface{}, e.channelLength)
	e.routes = append(e.routes, route{eui.String(), events})

	return events
}

// Unsubscribe from channel
func (e *EventRouter) Unsubscribe(ch <-chan interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for i, r := range e.routes {
		if r.ch == ch {
			close(r.ch)
			e.routes = append(e.routes[:i], e.routes[i+1:]...)
			break
		}
	}
}

// Publish publishes a gateway event to subscribers. If there are no subscribers
// the event will be ignored. If the event subscribers can't keep up with the events
// the events will be silently dropped.
func (e *EventRouter) Publish(eui protocol.EUI, event interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, route := range e.routes {
		if route.eui == eui.String() {
			select {
			case route.ch <- event:
				// This is OK
			default:
				logging.Warning("Channel client isn't keeping up with reads. Skipping the event (%v) for EUI %s", event, eui)
				// Skip event
			}
		}
	}
}
