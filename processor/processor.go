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
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/logging"
)

// Pipeline is the main processing pipeline for the server. Each step in
// the pipeline is handled by one or more goroutines. Channels are used to
// forward messages between the steps in the pipeline the channels are
// unbuffered at the moment and each step runs as a single goroutine. If one
// of the steps end up being a bottleneck we can increase the number of outputs
// and at the same time buffer the channels.
//
// The pipeline is roughly built like this:
//
//    GW Forwarder -> Decoder -> Decrypter -> MAC Processor
//          => Scheduler => Encoder -> GW Forwarder
//
type Pipeline struct {
	Decoder      *Decoder
	Decrypter    *Decrypter
	MACProcessor *MACProcessor
	Scheduler    *Scheduler
	Encoder      *Encoder
}

// Start launches the pipeline
func (p *Pipeline) Start() {
	go p.Decoder.Start()
	go p.Decrypter.Start()
	go p.MACProcessor.Start()
	go p.Scheduler.Start()
	go p.Encoder.Start()
}

// NewPipeline creates a new pipeline. The pipeline will stop automatically
// when the forwarder is terminated
func NewPipeline(context *server.Context, forwarder GwForwarder) *Pipeline {
	ret := Pipeline{}

	logging.Debug("Creating decoder...")
	ret.Decoder = NewDecoder(context, forwarder.Output())

	logging.Debug("Creating decrypter...")
	ret.Decrypter = NewDecrypter(context, ret.Decoder.Output())

	logging.Debug("Creating MAC processor...")
	ret.MACProcessor = NewMACProcessor(context, ret.Decrypter.Output())

	logging.Debug("Creating scheduler...")
	ret.Scheduler = NewScheduler(context, ret.MACProcessor.CommandNotifier())

	logging.Debug("Creating encoder...")
	ret.Encoder = NewEncoder(context, ret.Scheduler.Output(), forwarder.Input())

	return &ret
}
