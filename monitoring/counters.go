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
	"expvar"
)

type timeseriesCounter struct {
	name             string
	minuteTimeSeries *TimeSeries
	total            *expvar.Int
}

func (c *timeseriesCounter) init() {
	expvar.Publish(c.name+".minute", c.minuteTimeSeries)
	expvar.Publish(c.name+".total", c.total)
}

func (c *timeseriesCounter) Increment() {
	c.minuteTimeSeries.Increment()
	c.total.Add(1)
}
func newTimeseriesCounter(name string) *timeseriesCounter {
	ret := &timeseriesCounter{name, NewTimeSeries(Minutes), expvar.NewInt(name)}
	ret.init()
	return ret
}

type histogramCounter struct {
	name      string
	histogram *Histogram
	gauge     *AverageGauge
}

func (h *histogramCounter) init() {
	expvar.Publish(h.name+".histogram", h.histogram)
	expvar.Publish(h.name+".average", h.gauge)
}

// Add adds a new sample to the histogram counter
func (h *histogramCounter) Add(value float64) {
	h.histogram.Add(value)
	h.gauge.Add(value)
}

func newHistogramCounter(name string) *histogramCounter {
	ret := &histogramCounter{name, NewHistogram(), NewAverageGauge(1000)}
	ret.init()
	return ret
}

// These are the types of counters available. List is WIP
var (
	GatewayCreated      *timeseriesCounter
	GatewayUpdated      *timeseriesCounter
	GatewayRemoved      *timeseriesCounter
	ApplicationCreated  *timeseriesCounter
	ApplicationUpdated  *timeseriesCounter
	ApplicationRemoved  *timeseriesCounter
	DeviceCreated       *timeseriesCounter
	DeviceUpdated       *timeseriesCounter
	DeviceRemoved       *timeseriesCounter
	LoRaMICFailed       *timeseriesCounter
	LoRaConfirmedUp     *timeseriesCounter // Received by server
	LoRaConfirmedDown   *timeseriesCounter // Received by server
	LoRaUnconfirmedUp   *timeseriesCounter // Sent by server
	LoRaUnconfirmedDown *timeseriesCounter // Sent by server
	LoRaJoinRequest     *timeseriesCounter // Received by server
	LoRaJoinAccept      *timeseriesCounter // Sent by server
	LoRaCounterFailed   *timeseriesCounter // Rejected frame counter
	GatewayIn           *timeseriesCounter
	GatewayOut          *timeseriesCounter
	Decoder             *timeseriesCounter
	Decrypter           *timeseriesCounter
	MACProcessor        *timeseriesCounter
	SchedulerIn         *timeseriesCounter
	SchedulerOut        *timeseriesCounter
	Encoder             *timeseriesCounter

	GatewayChannelOut      *histogramCounter // Time to send message to decoder
	DecoderChannelOut      *histogramCounter // Time to send message to decrypter
	DecrypterChannelOut    *histogramCounter // Time to send message to MAC processor
	MACProcessorChannelOut *histogramCounter // Time to send message to scheduler
	SchedulerChannelOut    *histogramCounter // Time to send message to encoder
	EncoderChannelOut      *histogramCounter // Time to send message to gwid

	TimeGatewaySend      *histogramCounter
	TimeGatewayReceive   *histogramCounter
	TimeDecoder          *histogramCounter
	TimeDecrypter        *histogramCounter
	TimeEncoder          *histogramCounter
	TimeMACProcessor     *histogramCounter
	TimeSchedulerSend    *histogramCounter
	TimeSchedulerProcess *histogramCounter

	TimeIncoming *histogramCounter
	TimeOutgoing *histogramCounter

	MissedDeadline *timeseriesCounter
)

func init() {
	GatewayCreated = newTimeseriesCounter("gateway.create")
	GatewayUpdated = newTimeseriesCounter("gateway.update")
	GatewayRemoved = newTimeseriesCounter("gateway.delete")
	ApplicationCreated = newTimeseriesCounter("application.create")
	ApplicationUpdated = newTimeseriesCounter("application.update")
	ApplicationRemoved = newTimeseriesCounter("application.delete")
	DeviceCreated = newTimeseriesCounter("device.create")
	DeviceUpdated = newTimeseriesCounter("device.update")
	DeviceRemoved = newTimeseriesCounter("device.delete")
	LoRaMICFailed = newTimeseriesCounter("lora.mic.failed")
	LoRaCounterFailed = newTimeseriesCounter("lora.fcnt.failed")
	LoRaConfirmedUp = newTimeseriesCounter("lora.msg.confirmedup")
	LoRaConfirmedDown = newTimeseriesCounter("lora.msg.confirmeddown")
	LoRaUnconfirmedUp = newTimeseriesCounter("lora.msg.unconfirmedup")
	LoRaUnconfirmedDown = newTimeseriesCounter("lora.msg.unconfirmeddown")
	LoRaJoinRequest = newTimeseriesCounter("lora.msg.joinrequest")
	LoRaJoinAccept = newTimeseriesCounter("lora.msg.joinaccept")
	GatewayIn = newTimeseriesCounter("process.gateway.in")
	GatewayOut = newTimeseriesCounter("process.gateway.out")
	Decoder = newTimeseriesCounter("process.decoder")
	Decrypter = newTimeseriesCounter("process.decrypter")
	MACProcessor = newTimeseriesCounter("process.macprocessor")
	SchedulerIn = newTimeseriesCounter("process.scheduler.in")
	SchedulerOut = newTimeseriesCounter("process.scheduler.out")
	Encoder = newTimeseriesCounter("process.encoder")

	GatewayChannelOut = newHistogramCounter("gwif.channel.send")
	DecoderChannelOut = newHistogramCounter("decoder.channel.send")
	DecrypterChannelOut = newHistogramCounter("decrypter.channel.send")
	MACProcessorChannelOut = newHistogramCounter("macprocessor.channel.send")
	EncoderChannelOut = newHistogramCounter("encoder.channel.send")
	SchedulerChannelOut = newHistogramCounter("scheduler.channel.sends")

	TimeGatewaySend = newHistogramCounter("gateway.send.timing")
	TimeGatewayReceive = newHistogramCounter("gateway.receive.timing")
	TimeDecoder = newHistogramCounter("decoder.timing")
	TimeDecrypter = newHistogramCounter("decrypter.timing")
	TimeEncoder = newHistogramCounter("encoder.timing")
	TimeMACProcessor = newHistogramCounter("macprocessor.timing")
	TimeSchedulerSend = newHistogramCounter("scheduler.send.timing")
	TimeSchedulerProcess = newHistogramCounter("scheduler.process.timing")

	TimeIncoming = newHistogramCounter("incoming.timing")
	TimeOutgoing = newHistogramCounter("outgoing.timing")

	MissedDeadline = newTimeseriesCounter("process.deadlineMissed")
}
