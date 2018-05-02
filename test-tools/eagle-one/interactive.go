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
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	lassie "github.com/telenordigital/lassie-go"
)

type interactiveDevice struct {
	device   lassie.Device
	emulated *EmulatedDevice
}

// InteractiveMode is the interactive
type InteractiveMode struct {
	devices          []interactiveDevice
	selectedDevice   int
	congress         *lassie.Client
	Config           Params
	Publisher        *EventRouter
	OutgoingMessages chan string
	AppEUI           string
}

// Prepare prepares the console
func (i *InteractiveMode) Prepare(congress *lassie.Client, app lassie.Application, gw lassie.Gateway) error {
	// Nothing to do here
	i.congress = congress
	return nil
}

// Cleanup cleans up the console
func (i *InteractiveMode) Cleanup(congress *lassie.Client, app lassie.Application, gw lassie.Gateway) {
	if !i.Config.KeepDevices {
		for _, v := range i.devices {
			congress.DeleteDevice(app.EUI, v.device.EUI)
		}
	}
}

const yellowText = "\x1b[33;1m"
const whiteText = "\x1b[0m"

// Run starts the console
func (i *InteractiveMode) Run(outgoingMessages chan string, publisher *EventRouter, app lassie.Application, gw lassie.Gateway) {
	i.Publisher = publisher
	i.OutgoingMessages = outgoingMessages
	i.AppEUI = app.EUI
	time.Sleep(500 * time.Millisecond)
	fmt.Println("===============================================================================")
	fmt.Println("Eagle One interactive console")
	running := true
	reader := bufio.NewReader(os.Stdin)

	commands := []Command{
		&quitCommand{},
		&listCommand{i},
		&newDevice{app.EUI, i.congress, i},
		&removeDevice{app.EUI, i.congress, i},
		&selectDevice{i},
		&printDevice{i},
		&joinDevice{i},
		&sendMessage{i},
		&ackMessage{i},
	}
	for running {
		deviceEUI := "<none>"
		if len(i.devices) > i.selectedDevice {
			deviceEUI = i.devices[i.selectedDevice].device.EUI
		}
		fmt.Printf("%s[ 🦅  /applications/%s/devices/%s ]%s ", yellowText, i.AppEUI, deviceEUI, whiteText)
		line, _, _ := reader.ReadLine()
		found := false
		cmd := strings.TrimSpace(strings.ToLower(string(line)))
		if cmd == "help" || cmd == "h" || cmd == "?" {
			for _, c := range commands {
				fmt.Println(c.Help())
			}
			found = true
		}
		for _, c := range commands {
			if c.Matches(cmd) {
				params := strings.Split(cmd, " ")
				running = c.Execute(params[1:])
				found = true
				break
			}
		}
		if !found {
			fmt.Println("Unknown command: ", cmd)
		}
	}
}

// Failed returns true if the mode has failed
func (i *InteractiveMode) Failed() bool {
	return false
}
