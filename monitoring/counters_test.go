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
import "testing"

// Run through the test. Unless there are error messages in the test the counter
// will be available in the application
func TestCounterIncrement(t *testing.T) {
	GatewayCreated.Increment()
	GatewayUpdated.Increment()
	GatewayRemoved.Increment()
	ApplicationCreated.Increment()
	ApplicationUpdated.Increment()
	ApplicationRemoved.Increment()
	DeviceCreated.Increment()
	DeviceUpdated.Increment()
	DeviceRemoved.Increment()
	LoRaMICFailed.Increment()
	LoRaConfirmedUp.Increment()
	LoRaConfirmedDown.Increment()
	LoRaUnconfirmedUp.Increment()
	LoRaUnconfirmedDown.Increment()
	LoRaJoinRequest.Increment()
	LoRaJoinAccept.Increment()
	LoRaCounterFailed.Increment()
	GatewayIn.Increment()
	GatewayOut.Increment()
	Decoder.Increment()
	Decrypter.Increment()
	MACProcessor.Increment()
	SchedulerIn.Increment()
	SchedulerOut.Increment()
	Encoder.Increment()

	GatewayChannelOut.Add(1.0)
	DecoderChannelOut.Add(1.0)
	DecrypterChannelOut.Add(1.0)
	MACProcessorChannelOut.Add(1.0)
	SchedulerChannelOut.Add(1.0)
	EncoderChannelOut.Add(1.0)

	TimeGatewaySend.Add(2.0)
	TimeGatewayReceive.Add(2.0)
	TimeDecoder.Add(2.0)
	TimeDecrypter.Add(2.0)
	TimeEncoder.Add(2.0)
	TimeMACProcessor.Add(2.0)
	TimeSchedulerSend.Add(2.0)
	TimeSchedulerProcess.Add(2.0)

	TimeIncoming.Add(3.0)
	TimeOutgoing.Add(3.0)
}
