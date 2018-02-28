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
	"math/rand"
	"testing"
	"time"

	"sync"

	"github.com/ExploratoryEngineering/congress/events/gwevents"
	"github.com/ExploratoryEngineering/congress/protocol"
)

// Simple one-shot route test
func TestEventRouter(t *testing.T) {

	router := NewEventRouter(2)

	ch := router.Subscribe(protocol.EUIFromUint64(0))

	router.Publish(protocol.EUIFromUint64(0), gwevents.NewInactive())

	select {
	case <-ch:
		// This is ok
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Didn't get an event on the channel")
	}

	router.Unsubscribe(ch)
}

// Test with multiple routes (and channels)
func TestEventRouterMultipleRoutes(t *testing.T) {
	const numEvents = 4
	router := NewEventRouter(numEvents)
	wg := sync.WaitGroup{}

	const routes = 10
	euis := make([]protocol.EUI, routes)
	for i := 0; i < routes; i++ {
		euis[i] = protocol.EUIFromUint64(uint64(i))
	}

	chans := make([]<-chan interface{}, routes)

	for i := 0; i < routes; i++ {
		chans[i] = router.Subscribe(euis[i])
	}

	wg.Add(routes)
	for _, ch := range chans {
		ch := ch
		go func() {
			received := 0
			for {
				select {
				case <-ch:
					received++
					if received == numEvents {
						wg.Done()
						return
					}
				case <-time.After(100 * time.Millisecond):
					t.Fatalf("Didn't receive data! Got just %d events, expected 5", received)
				}
			}
		}()
	}

	publish := func() {
		for i := 0; i < routes; i++ {
			router.Publish(euis[i], gwevents.NewKeepAlive())
			router.Publish(euis[i], gwevents.NewKeepAlive())
			router.Publish(euis[i], gwevents.NewTx("some data"))
			router.Publish(euis[i], gwevents.NewRx("some data"))
		}
	}

	publish()

	wg.Wait()

	for i := routes - 1; i >= 0; i-- {
		router.Unsubscribe(chans[i])
	}

	publish()
}

// Create multiple copies of the same subscription and size up and down. The
// output isn't *that* interesting; the test just ensures edge cases aren't missed.
func TestResize(t *testing.T) {
	const routeCount = 100
	router := NewEventRouter(2)

	var subs []<-chan interface{}

	eui := protocol.EUIFromUint64(4711)
	for i := 0; i < routeCount; i++ {
		ch := router.Subscribe(eui)
		subs = append(subs, ch)
	}

	// Publish one
	router.Publish(eui, gwevents.NewInactive())

	for i := 0; i < routeCount/2; i++ {
		router.Unsubscribe(subs[rand.Int()%routeCount])
	}

	router.Publish(eui, gwevents.NewKeepAlive())

	for i := 0; i < routeCount; i++ {
		router.Unsubscribe(subs[i])
	}
}
