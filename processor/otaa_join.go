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
	"github.com/ExploratoryEngineering/congress/frequency"
	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/monitoring"
	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/logging"
)

// Process the join request. Returns false if it failed.
func (d *Decrypter) processJoinRequest(decoded server.LoRaMessage) bool {
	monitoring.LoRaJoinRequest.Increment()
	joinRequest := &decoded.Payload.JoinRequestPayload

	device, err := d.context.Storage.Device.GetByEUI(joinRequest.DevEUI)
	if err != nil {
		logging.Info("Unknown device attempting JoinRequest: %s", joinRequest.DevEUI)
		return false
	}

	if device.AppEUI != joinRequest.AppEUI {
		logging.Warning("Mismatch between stored device's AppEUI and the AppEUI sent in the JoinRequest message. Stored AppEUI = %s, JoinRequest AppEUI = %s", device.AppEUI, joinRequest.AppEUI)
		return false
	}

	// Check if DevNonce have been used by the device in an earlier request.
	// If so the request should be ignored. [6.2.4].
	if device.HasDevNonce(joinRequest.DevNonce) {
		logging.Warning("Device %s has already used nonce 0x%04x. Ignoring it.",
			joinRequest.DevEUI, joinRequest.DevNonce)
		return false
	}

	// Retrieve the application
	app, err := d.context.Storage.Application.GetByEUI(joinRequest.AppEUI, model.SystemUserID)
	if err != nil {
		logging.Warning("Unable to retrieve application with EUI %s. Ignoring JoinRequest from device with EUI %s",
			joinRequest.AppEUI, joinRequest.DevEUI)
		return false
	}

	// Invariant: Application is OK, device is OK, network is OK. Generate response
	decoded.FrameContext.Application = app
	decoded.FrameContext.Device = device

	// DevAddr is already assigned to the device. It is a function of the EUI.

	// Update the device with new keys and DevNonce
	if err := d.context.Storage.Device.AddDevNonce(device, joinRequest.DevNonce); err != nil {
		logging.Warning("Unable to update DevNonce on device with EUI: %s: %v",
			device.DeviceEUI, err)
	}

	// Generate app nonce, generate keys, store keys
	appNonce, err := app.GenerateAppNonce()
	if err != nil {
		logging.Warning("Unable to generate app nonce: %v (devEUI: %s, appEUI: %s). Ignoring JoinRequest",
			err, joinRequest.DevEUI, joinRequest.AppEUI)
		return false
	}
	nwkSKey, err := protocol.NwkSKeyFromNonces(device.AppKey, appNonce, uint32(d.context.Config.NetworkID), joinRequest.DevNonce)
	if err != nil {
		logging.Error("Unable to generate NwkSKey for device with EUI %s: %v", device.DeviceEUI, err)
		return false
	}
	appSKey, err := protocol.AppSKeyFromNonces(device.AppKey, appNonce, uint32(d.context.Config.NetworkID), joinRequest.DevNonce)
	if err != nil {
		logging.Error("Unable to generate AppSKey for device with EUI %s: %v", device.DeviceEUI, err)
		return false
	}
	device.NwkSKey = nwkSKey
	device.AppSKey = appSKey
	device.FCntDn = 0
	device.FCntUp = 0
	if err := d.context.Storage.Device.Update(device); err != nil {
		logging.Error("Unable to update device with EUI %s: %v", device.DeviceEUI, err)
		return false
	}

	// Invariant. Everything is OK - make a JoinAccept response and schedule
	// the output.
	joinAccept := protocol.JoinAcceptPayload{
		AppNonce:   appNonce,
		NetID:      uint32(d.context.Config.NetworkID),
		DevAddr:    device.DevAddr,
		DLSettings: frequency.GetDLSettingsOTAA(),
		RxDelay:    frequency.GetRxDelayOTAA(),
		CFList:     frequency.GetCFListOTAA(),
	}

	d.context.FrameOutput.SetJoinAcceptPayload(device.DeviceEUI, joinAccept)

	logging.Debug("JoinAccept sent to %s. DevAddr=%$", device.DeviceEUI, joinAccept.DevAddr)

	// The incoming message doesn't have a DevAddr set but schedule an empty
	// message for it. TODO (stalehd): this is butt ugly. Needs redesign.
	decoded.Payload.MACPayload.FHDR.DevAddr = joinAccept.DevAddr

	decoded.FrameContext.GatewayContext.SectionTimer.End()
	d.macOutput <- decoded
	monitoring.LoRaJoinAccept.Increment()
	return true
}
