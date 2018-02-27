package main
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
import lassie "github.com/telenordigital/lassie-go"

// E1Mode is the E1 modes
type E1Mode interface {
	Prepare(congress *lassie.Client, app lassie.Application, gw lassie.Gateway) error
	Cleanup(congress *lassie.Client, app lassie.Application, gw lassie.Gateway)
	Run(gatewayChannel chan string, publisher *EventRouter, app lassie.Application, gw lassie.Gateway)
	Failed() bool
}
