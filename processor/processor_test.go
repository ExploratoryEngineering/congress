package processor

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
	"crypto/rand"
	"reflect"
	"testing"
	"time"

	"github.com/ExploratoryEngineering/congress/band"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/congress/storage/memstore"
	"github.com/ExploratoryEngineering/logging"
)

// Test methods. Simplifies setting up test context

// TestAppEUI is the app EUI used for test storage context
var TestAppEUI protocol.EUI

func init() {
	TestAppEUI, _ = protocol.EUIFromString("74-0A-09-3A-2C-22-69-F3")

}

// NewStorageTestContext creates a new storage context. This is populated with dummy data.
func NewStorageTestContext() storage.Storage {

	application := model.Application{
		AppEUI: TestAppEUI,
		Tags:   model.NewTags(),
	}

	// EUI: 750A093A2C2269F3
	deviceEUI, _ := protocol.EUIFromString("75-0A-09-3A-2C-22-69-F3")
	appSKey, _ := protocol.AESKeyFromString("E001 2A22 25B8 585E DCEC 7042 4798 C510")
	nwkSKey, _ := protocol.AESKeyFromString("3C5E 5C9F 469E EF3E 02CC D4FF 9531 31BA")
	device := model.Device{
		DeviceEUI: deviceEUI,
		DevAddr: protocol.DevAddr{
			NwkID:   0,
			NwkAddr: 0x1E672E6,
		},
		AppSKey:        appSKey,
		NwkSKey:        nwkSKey,
		AppEUI:         application.AppEUI,
		State:          model.PersonalizedDevice,
		FCntUp:         0,
		FCntDn:         0,
		RelaxedCounter: true,
	}

	store := memstore.CreateMemoryStorage(0, 0)

	if err := store.Application.Put(application, model.SystemUserID); err != nil {
		logging.Error("Could not add application to storage: ", err)
	}

	store.Device.Put(device, application.AppEUI)

	return store
}

// testForwarder is a debugging forwarder that just exposes the input and output channels
type testForwarder struct {
	input  chan server.GatewayPacket
	output chan server.GatewayPacket
}

// Inject a message into the pipeline
func (t *testForwarder) injectMessage(pkt server.GatewayPacket) {
	t.output <- pkt
}

// Grab a message destined for the forwarder. Waits for timeout time and if the error message is set the test fails
func (t *testForwarder) grabMessage(timeout time.Duration) *server.GatewayPacket {
	select {
	case m := <-t.input:
		return &m
	case <-time.After(timeout):
		return nil
	}
}

func (t *testForwarder) Start() {
}

func (t *testForwarder) Stop() {
	close(t.input)
	close(t.output)
}

func (t *testForwarder) Input() chan<- server.GatewayPacket {
	return t.input
}

func (t *testForwarder) Output() <-chan server.GatewayPacket {
	return t.output
}

func newTestForwarder() *testForwarder {
	return &testForwarder{
		input:  make(chan server.GatewayPacket),
		output: make(chan server.GatewayPacket),
	}
}

// Helper method to build PHYPayload message
func newPHYPayloadMessage(messageType protocol.MType, devAddr protocol.DevAddr) *protocol.PHYPayload {
	// Bypass the forwarder and emulate a new message into the pipeline
	ret := protocol.NewPHYPayload(messageType)
	ret.MACPayload.FHDR.DevAddr = devAddr
	ret.MACPayload.FPort = 1
	ret.MACPayload.FRMPayload = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	return &ret
}

var sendTime = time.Now()

func sendMessageOnChannel(c *testContext, msg *protocol.PHYPayload, device model.Device) {
	band, _ := band.NewBand(band.EU868Band)
	gwInput := server.GatewayPacket{
		Radio: server.RadioContext{ // Radio context is required to determine packet forwarder settings
			Band:     band,
			DataRate: "SF7BW125",
		},
		Gateway: server.GatewayContext{
			GatewayEUI: protocol.EUIFromUint64(0x0101010102020202),
		},
		ReceivedAt: c.messageTs,
	}
	gwInput.RawMessage, _ = msg.EncodeMessage(device.NwkSKey, device.AppSKey)
	c.forwarder.injectMessage(gwInput)
	// Increment the received time artificially to avoid storage conflicts
	c.messageTs = c.messageTs.Add(1 * time.Second)
}

type testContext struct {
	config    *server.Configuration
	context   *server.Context
	datastore storage.Storage
	app       model.Application
	device    model.Device
	forwarder *testForwarder
	pipeline  *Pipeline
	messageTs time.Time
	t         *testing.T
}

func newTestContext(t *testing.T) testContext {
	ret := testContext{t: t}
	ret.config = server.NewDefaultConfig()
	ret.config.MemoryDB = true
	ret.datastore = memstore.CreateMemoryStorage(0, 0)
	frameOutput := server.NewFrameOutputBuffer()
	keyGenerator, _ := server.NewEUIKeyGenerator(ret.config.RootMA(), uint32(ret.config.NetworkID), ret.datastore.Sequence)

	appRouter := server.NewEventRouter(5)
	gwEventRouter := server.NewEventRouter(5)
	ret.context = &server.Context{
		Storage:       &ret.datastore,
		Terminator:    make(chan bool),
		FrameOutput:   &frameOutput,
		Config:        ret.config,
		KeyGenerator:  &keyGenerator,
		GwEventRouter: &gwEventRouter,
		AppRouter:     &appRouter,
		AppOutput:     server.NewAppOutputManager(&appRouter),
	}
	ret.forwarder = newTestForwarder()
	ret.pipeline = NewPipeline(ret.context, ret.forwarder)

	ret.app = model.NewApplication()
	ret.app.AppEUI, _ = keyGenerator.NewAppEUI()

	ret.datastore.Application.Put(ret.app, model.SystemUserID)

	ret.device = model.NewDevice()
	ret.device.DeviceEUI, _ = keyGenerator.NewDeviceEUI()
	ret.device.AppEUI = ret.app.AppEUI
	ret.device.DevAddr = protocol.DevAddrFromUint32(0x00112233)
	ret.device.AppKey, _ = protocol.NewAESKey()
	ret.device.AppSKey, _ = protocol.NewAESKey()
	ret.device.NwkSKey, _ = protocol.NewAESKey()
	ret.device.State = model.PersonalizedDevice
	ret.device.RelaxedCounter = true

	ret.datastore.Device.Put(ret.device, ret.app.AppEUI)

	return ret
}

func checkMessageOutput(c *testContext, testCase string, checkMessage func(packet protocol.PHYPayload)) {
	msg := c.forwarder.grabMessage(1 * time.Second)
	if msg == nil {
		c.t.Fatalf("Did not get a response within the 1 second limit (test %s)", testCase)
	}
	phy := protocol.NewPHYPayload(protocol.Proprietary)
	if err := phy.UnmarshalBinary(msg.RawMessage); err != nil {
		c.t.Fatalf("Unable to unmarshal packet going to gateway: %v (test %s)", err, testCase)
	}
	checkMessage(phy)
}

// Tt.his is a *big* test of the pipeline. To test the downstream message processing fully we have to
// tt.o what amounts to a full integration test between all of the modules. The test itself is fairly
// simple -- set up storage, create a pipeline and pass messages through it emulating devices
// that send messages to the server.
func TestProcessingPipeline(t *testing.T) {
	const timeToWaitForNoMessage = 20 * time.Millisecond

	c := newTestContext(t)
	c.pipeline.Scheduler.SetRXDelay(5 * time.Millisecond)
	c.pipeline.Start()

	// ----------------------------------------------------------------------
	// Test 1: ConfirmedDataUp from device, no downstream message waiting
	//     => mtype=UnconfirmedDown, port=0, ack=true
	sendMessageOnChannel(&c, newPHYPayloadMessage(protocol.ConfirmedDataUp, c.device.DevAddr), c.device)

	checkMessageOutput(&c, "1", func(phy protocol.PHYPayload) {
		if phy.MHDR.MType != protocol.UnconfirmedDataDown {
			t.Fatal("Expected unconfirmed data down")
		}
		if phy.MACPayload.FPort != 0 {
			t.Fatalf("Expected port 0 but got port %d", phy.MACPayload.FPort)
		}
		if !phy.MACPayload.FHDR.FCtrl.ACK {
			t.Fatalf("Server should ack message")
		}
	})

	// ----------------------------------------------------------------------
	// Test 2: UnconfirmedUp from device, no downstream message
	//     => no response
	sendMessageOnChannel(&c, newPHYPayloadMessage(protocol.UnconfirmedDataUp, c.device.DevAddr), c.device)
	select {
	case <-c.forwarder.Output():
		t.Fatalf("Did not expect a message to be sent in response to UnconfirmedUp message")
	case <-time.After(timeToWaitForNoMessage):
		// this is OK
	}

	// ----------------------------------------------------------------------
	// Test 3: UnconfirmedUp from device, unconfirmed downstream message
	//    => downstream message with no ack
	downMsg := model.NewDownstreamMessage(c.device.DeviceEUI, 200)
	downMsg.Ack = false
	downMsg.Data = "010203040506070809"
	if err := c.datastore.DeviceData.PutDownstream(c.device.DeviceEUI, downMsg); err != nil {
		t.Fatalf("Unable to store downstream: %v", err)
	}

	sendMessageOnChannel(&c, newPHYPayloadMessage(protocol.UnconfirmedDataUp, c.device.DevAddr), c.device)

	checkMessageOutput(&c, "3", func(phy protocol.PHYPayload) {
		if phy.MHDR.MType != protocol.UnconfirmedDataDown {
			t.Fatal("Expected unconfirmed data down")
		}
		if phy.MACPayload.FPort != downMsg.Port {
			t.Fatalf("Expected port %d but got port %d", downMsg.Port, phy.MACPayload.FPort)
		}
		if phy.MACPayload.FHDR.FCtrl.ACK {
			t.Fatalf("Server shouldn't ack message")
		}
		phy.Decrypt(c.device.NwkSKey, c.device.AppSKey)
		if !reflect.DeepEqual(phy.MACPayload.FRMPayload, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}) {
			t.Fatalf("Did not get the expected data: %v", phy.MACPayload.FRMPayload)
		}
	})
	downMsg, err := c.datastore.DeviceData.GetDownstream(c.device.DeviceEUI)
	if err != nil {
		t.Fatalf("Unable to retrieve downstream message: %v", err)
	}
	if downMsg.SentTime == 0 {
		t.Fatal("SentTime should be set when message is sent but it is 0")
	}

	// ----------------------------------------------------------------------
	// Test 3a: UnconfirmedUp from device, already sent downstream message
	//    => no response
	sendMessageOnChannel(&c, newPHYPayloadMessage(protocol.UnconfirmedDataUp, c.device.DevAddr), c.device)
	if msg := c.forwarder.grabMessage(timeToWaitForNoMessage); msg != nil {
		t.Fatalf("Did not expect a response from the server but got %v", msg)
	}

	// ----------------------------------------------------------------------
	// Test 4: UnconfirmedUp from device, downstream message w/ ack
	//    => Downstream message
	downMsgAck := model.NewDownstreamMessage(c.device.DeviceEUI, 100)
	downMsgAck.Ack = true
	downMsgAck.Data = "aabbccddeeff00112233"
	c.datastore.DeviceData.DeleteDownstream(c.device.DeviceEUI)
	if err := c.datastore.DeviceData.PutDownstream(c.device.DeviceEUI, downMsgAck); err != nil {
		t.Fatalf("Unable to store downstream message: %v", err)
	}
	sendMessageOnChannel(&c, newPHYPayloadMessage(protocol.UnconfirmedDataUp, c.device.DevAddr), c.device)
	checkMessageOutput(&c, "4", func(phy protocol.PHYPayload) {
		if phy.MHDR.MType != protocol.ConfirmedDataDown {
			t.Fatal("Did not get ConfirmedDataDown from server")
		}
		if phy.MACPayload.FPort != downMsgAck.Port {
			t.Fatalf("Got port %d but expected %d", phy.MACPayload.FPort, downMsgAck.Port)
		}
	})

	// ----------------------------------------------------------------------
	// Test 4a: UnconfirmedUp from device w/ no ack, same downstream message
	//    => Downstream message repeated
	sendMessageOnChannel(&c, newPHYPayloadMessage(protocol.UnconfirmedDataUp, c.device.DevAddr), c.device)
	checkMessageOutput(&c, "4a", func(phy protocol.PHYPayload) {
		if phy.MHDR.MType != protocol.ConfirmedDataDown {
			t.Fatal("Did not get ConfirmedDataDown from server")
		}
		if phy.MACPayload.FPort != downMsgAck.Port {
			t.Fatalf("Got port %d but expected %d", phy.MACPayload.FPort, downMsgAck.Port)
		}
	})

	// ----------------------------------------------------------------------
	// Test 4b: UnconfirmedUp from device w/ ack, same downstream message
	//    => no message but downstream updated w/ ack
	msg := newPHYPayloadMessage(protocol.UnconfirmedDataUp, c.device.DevAddr)
	msg.MACPayload.FHDR.FCtrl.ACK = true
	sendMessageOnChannel(&c, msg, c.device)

	if msg := c.forwarder.grabMessage(timeToWaitForNoMessage); msg != nil {
		t.Fatalf("Did not expect downstream message to be sent a 2nd time but got %v", msg)
	}
	updatedAckMsg, err := c.datastore.DeviceData.GetDownstream(c.device.DeviceEUI)
	if err != nil {
		t.Fatalf("Unable to retrieve downstream message: %v", err)
	}
	if updatedAckMsg.AckTime == 0 {
		t.Fatal("AckTime should be set but it wasn't")
	}
	oldAck := updatedAckMsg.AckTime

	// ----------------------------------------------------------------------
	// Test 4b: UnconfirmedUp from device w/ ack, same downstream message
	//    => no message
	msg = newPHYPayloadMessage(protocol.UnconfirmedDataUp, c.device.DevAddr)
	msg.MACPayload.FHDR.FCtrl.ACK = true
	sendMessageOnChannel(&c, msg, c.device)

	if msg := c.forwarder.grabMessage(timeToWaitForNoMessage); msg != nil {
		t.Fatalf("Did not expect downstream message to be sent a 3rd time but got %v", msg)
	}

	updatedAckMsg, err = c.datastore.DeviceData.GetDownstream(c.device.DeviceEUI)
	if err != nil {
		t.Fatalf("Unable to retrieve downstream message: %v", err)
	}
	if updatedAckMsg.AckTime != oldAck {
		t.Fatal("AckTime should not be updated")
	}

	// ----------------------------------------------------------------------
	// Test 5: The raison d'etre: Persisting messages when restaring
	// Add downstream message, shut down pipeline (in effect stopping the server),
	// launch a new pipeline and see if the message is forwarded appropriately.

	c.datastore.DeviceData.DeleteDownstream(c.device.DeviceEUI)
	persistedMsg := model.NewDownstreamMessage(c.device.DeviceEUI, 50)
	persistedMsg.Ack = true
	persistedMsg.Data = "beefbeefbeefbeef"

	if err := c.datastore.DeviceData.PutDownstream(c.device.DeviceEUI, persistedMsg); err != nil {
		t.Fatalf("Unable to store downstream message: %v", err)
	}

	// Test 5a: Send one message to the server. The message should be sent back
	sendMessageOnChannel(&c, msg, c.device)
	checkMessageOutput(&c, "5a", func(p protocol.PHYPayload) {
		if p.MHDR.MType != protocol.ConfirmedDataDown {
			t.Fatalf("Expected ConfirmedDataDown but didn't get it. Got %v.", p.MHDR.MType)
		}
	})
	c.forwarder.Stop()

	updatedMsg, err := c.datastore.DeviceData.GetDownstream(c.device.DeviceEUI)
	if err != nil {
		t.Fatalf("Error retrieving downstream msg: %v", err)
	}
	// Message is sent but not acknowledged
	if updatedMsg.State() != model.SentState {
		t.Fatalf("Unexpected state for downstream message. Expected SentState but got %v", updatedMsg.State())
	}

	// Start a totally new pipeline
	c.forwarder = newTestForwarder()
	c.pipeline = NewPipeline(c.context, c.forwarder)
	c.pipeline.Start()
	c.forwarder.Start()

	// Test 5b: Send a new message with no ack. The message should be resent
	sendMessageOnChannel(&c, newPHYPayloadMessage(protocol.UnconfirmedDataUp, c.device.DevAddr), c.device)
	checkMessageOutput(&c, "5b", func(p protocol.PHYPayload) {
		if p.MHDR.MType != protocol.ConfirmedDataDown {
			t.Fatalf("Got message type %v. Didn't expect that.", p.MHDR.MType)
		}
	})

	// Test 5c: Ack the message. No message will be receive and the message
	// will have changed state to acknowledged
	msg = newPHYPayloadMessage(protocol.UnconfirmedDataUp, c.device.DevAddr)
	msg.MACPayload.FHDR.FCtrl.ACK = true
	sendMessageOnChannel(&c, msg, c.device)

	if msg := c.forwarder.grabMessage(timeToWaitForNoMessage); msg != nil {
		t.Fatalf("Did not expect downstream message to be sent a 3rd time but got %v", msg)
	}

	updatedMsg, err = c.datastore.DeviceData.GetDownstream(c.device.DeviceEUI)
	if err != nil {
		t.Fatalf("Error retrieving downstream msg: %v", err)
	}
	// Message is sent but not acknowledged
	if updatedMsg.State() != model.AcknowledgedState {
		t.Fatalf("Unexpected state for downstream message. Expected SentState but got %v", updatedMsg.State())
	}

	c.forwarder.Stop()
}

// Simple test: Start and shut down pipeline
func TestPipelineUpDown(t *testing.T) {
	c := newTestContext(t)
	c.pipeline.Start()
	<-time.After(100 * time.Millisecond)
	c.forwarder.Stop()
	<-time.After(100 * time.Millisecond)

}

func makeRandomEUI() protocol.EUI {
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	ret := protocol.EUI{}
	copy(ret.Octets[:], randomBytes)
	return ret
}

func makeRandomDevice(appEUI protocol.EUI) model.Device {
	d := model.NewDevice()
	d.DeviceEUI = makeRandomEUI()
	d.AppEUI = appEUI
	d.AppKey, _ = protocol.NewAESKey()
	d.NwkSKey, _ = protocol.NewAESKey()
	d.AppSKey, _ = protocol.NewAESKey()
	d.DevAddr = protocol.NewDevAddr()
	d.RelaxedCounter = true
	d.State = model.PersonalizedDevice
	return d
}

// Test how duplicate DevAddr (but with unique NwkSKey/AppSKeys) is routed. The
// device with the matching keys should receive the response.
//
// Create one app with one device each. The devices have the same DevAddr
// but different keys. Emulate one message from each of the devices and
// make sure the correct device gets updated.
func TestDuplicateDevAddr(t *testing.T) {
	const timeToWaitForNoMessage = 20 * time.Millisecond

	c := newTestContext(t)
	c.pipeline.Scheduler.SetRXDelay(5 * time.Millisecond)
	c.pipeline.Start()

	app1 := model.NewApplication()
	app2 := model.NewApplication()
	app1.AppEUI = makeRandomEUI()
	app2.AppEUI = makeRandomEUI()

	c.datastore.Application.Put(app1, model.SystemUserID)
	c.datastore.Application.Put(app2, model.SystemUserID)

	device1 := makeRandomDevice(app1.AppEUI)
	device2 := makeRandomDevice(app2.AppEUI)
	device2.DevAddr = device1.DevAddr

	c.datastore.Device.Put(device1, app1.AppEUI)
	c.datastore.Device.Put(device2, app2.AppEUI)

	msg1 := newPHYPayloadMessage(protocol.ConfirmedDataUp, device1.DevAddr)
	msg1.MACPayload.FRMPayload = []byte{1, 1, 1, 1}
	sendMessageOnChannel(&c, msg1, device1)

	checkMessageOutput(&c, "Device 1", func(p protocol.PHYPayload) {
		if p.MHDR.MType != protocol.UnconfirmedDataDown {
			t.Fatalf("Got message type %v. Didn't expect that.", p.MHDR.MType)
		}
	})
	msg2 := newPHYPayloadMessage(protocol.ConfirmedDataUp, device2.DevAddr)
	msg2.MACPayload.FRMPayload = []byte{2, 2, 2, 2}
	sendMessageOnChannel(&c, msg2, device2)
	checkMessageOutput(&c, "Device 2", func(p protocol.PHYPayload) {
		if p.MHDR.MType != protocol.UnconfirmedDataDown {
			t.Fatalf("Got message type %v. Didn't expect that.", p.MHDR.MType)
		}
	})

	// Data for device 1 should contain *one* packet with payload "01010101"
	ch, err := c.datastore.DeviceData.GetByDeviceEUI(device1.DeviceEUI, 1)
	if err != nil {
		t.Fatalf("Got error retrieving data for device 1: %v", err)
	}
	count := 0
	for data := range ch {
		if !reflect.DeepEqual(data.Data, []byte{1, 1, 1, 1}) {
			t.Fatalf("Device data does not contain 1,1,1,1 but %v", data.Data)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("Got %d data elements, but expected 1", count)
	}
	// Data for deviec 2 should contain *one* packet with payload "02020202"
	ch, err = c.datastore.DeviceData.GetByDeviceEUI(device2.DeviceEUI, 1)
	if err != nil {
		t.Fatalf("Got error retrieving data for device 2: %v", err)
	}
	count = 0
	for data := range ch {
		if !reflect.DeepEqual(data.Data, []byte{2, 2, 2, 2}) {
			t.Fatalf("Device data does not contain 2,2,2,2 but %v", data.Data)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("Got %d data elements, but expected 1", count)
	}

	<-time.After(100 * time.Millisecond)
	c.forwarder.Stop()
}

// Test decoding with invalid MIC
func TestInvalidMIC(t *testing.T) {
	const timeToWaitForNoMessage = 80 * time.Millisecond
	c := newTestContext(t)
	c.pipeline.Scheduler.SetRXDelay(5 * time.Millisecond)
	c.pipeline.Start()
	newDevice := c.device
	newDevice.NwkSKey, _ = protocol.NewAESKey()
	newDevice.AppSKey, _ = protocol.NewAESKey()
	sendMessageOnChannel(&c, newPHYPayloadMessage(protocol.ConfirmedDataUp, c.device.DevAddr), newDevice)

	if msg := c.forwarder.grabMessage(timeToWaitForNoMessage); msg != nil {
		t.Fatalf("Did not expect an ack message %v", msg)
	}
	c.forwarder.Stop()
}
