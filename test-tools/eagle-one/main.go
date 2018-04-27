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
import (
	"fmt"
	"net/url"
	"os"

	"github.com/ExploratoryEngineering/logging"
	"github.com/telenordigital/lassie-go"
)

func main() {
	logging.EnableStderr(false)
	if err := CommandLineParameters.Valid(); err != nil {
		logging.Error("Invalid configuration: %v", err)
		os.Exit(1)
	}
	logging.SetLogLevel(uint(CommandLineParameters.LogLevel))

	var mode E1Mode
	switch CommandLineParameters.Mode {
	case "batch":
		mode = &BatchMode{Config: CommandLineParameters}
	case "interactive":
		mode = &InteractiveMode{Config: CommandLineParameters}
	case "test":
		mode = &TestMode{Config: CommandLineParameters}
	default:
		logging.Error("Unknown mode: " + CommandLineParameters.Mode)
		os.Exit(1)
	}

	congress, err := lassie.New()
	if err != nil {
		logging.Error("Couldn't create the Congress API client: %v", err)
		os.Exit(1)
	}
	u, err := url.Parse(congress.Address())
	if err != nil {
		logging.Error("Invalid Congress URL: %v", err)
		os.Exit(1)
	}
	CommandLineParameters.Hostname = u.Hostname()

	logging.Info("Congress UDP: %s:%d", u.Hostname(), CommandLineParameters.UDPPort)
	logging.Info("Using Congress API at: %s", congress.Address())

	e1 := Eagle1{
		Congress:       congress,
		Config:         CommandLineParameters,
		Publisher:      NewEventRouter(2),
		GatewayChannel: make(chan string, 2),
	}
	if err := e1.Setup(); err != nil {
		logging.Error("Init error: %v", err)
		os.Exit(1)
	}
	defer e1.Teardown()

	e1.StartForwarder()

	if err := e1.Run(mode); err != nil {
		logging.Error("Unable to run %s mode: %v", CommandLineParameters.Mode, err)
		os.Exit(1)
	}

	if mode.Failed() {
		fmt.Println("Exiting with errors")
		os.Exit(1)
	}
	fmt.Println("Successful stop")
}
