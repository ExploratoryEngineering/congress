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
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/ExploratoryEngineering/congress/protocol"
	lassie "github.com/telenordigital/lassie-go"
)

// Command is an interactive command implementation
type Command interface {
	Matches(string) bool
	Execute(params []string) bool
	Help() string
}

func helpText(cmds, description, params string) string {
	const helpAlign = 40
	cmdsize := len(cmds)
	padding := strings.Repeat(".", helpAlign-cmdsize)
	return fmt.Sprintf("%s%s: %s %s", cmds, padding, description, params)
}

// -------------------------------------------------
// quit implements
type quitCommand struct {
}

func (q *quitCommand) Matches(cmd string) bool {
	return strings.HasPrefix(cmd, "q") || strings.HasPrefix(cmd, "exit") || strings.HasPrefix(cmd, "x")
}

func (q *quitCommand) Execute(params []string) bool {
	return false
}

func (q *quitCommand) Help() string {
	return helpText("q, quit, exit", "exit console", "")
}

// -------------------------------------------------
// list devices
type listCommand struct {
	mode *InteractiveMode
}

func (l *listCommand) Matches(cmd string) bool {
	return strings.HasPrefix(cmd, "l")
}

func (l *listCommand) Execute(params []string) bool {
	fmt.Println()
	fmt.Printf("Sel  #     Device EUI               Type\n")
	fmt.Printf("---  ----  -----------------------  ----\n")
	for i, d := range l.mode.devices {
		activestr := "   "
		if l.mode.selectedDevice == i {
			activestr = " * "
		}
		fmt.Printf("%s  %4d  %s  %s\n", activestr, i, d.device.EUI, d.device.Type)
	}
	fmt.Println()
	return true
}

func (l *listCommand) Help() string {
	return helpText("l, ls", "list devices", "")

}

// -------------------------------------------------
// new device
type newDevice struct {
	appEUI   string
	congress *lassie.Client
	mode     *InteractiveMode
}

func (n *newDevice) Matches(cmd string) bool {
	return strings.HasPrefix(cmd, "n")
}

func (n *newDevice) Execute(params []string) bool {
	newDevice := lassie.Device{}
	for _, v := range params {
		keyval := strings.Split(v, "=")
		if len(keyval) != 2 {
			continue
		}
		switch strings.ToLower(keyval[0]) {
		case "otaa":
			newDevice.Type = "OTAA"
		case "abp":
			newDevice.Type = "ABP"
		case "nwkskey":
			newDevice.NetworkSessionKey = keyval[1]
		case "appskey":
			newDevice.ApplicationSessionKey = keyval[1]
		case "type":
			newDevice.Type = keyval[1]
		case "relaxedcounter":
			newDevice.RelaxedCounter = (keyval[1] == "true" || keyval[1] == "1")
		case "appkey":
			newDevice.ApplicationKey = keyval[1]
		case "devaddr":
			newDevice.DeviceAddress = keyval[1]
		case "eui":
			newDevice.EUI = keyval[1]
		case "fcntup":
			fc, err := strconv.Atoi(keyval[1])
			if err != nil {
				fmt.Println("Invalid value for fcntup: ", keyval[1])
				return true
			}
			newDevice.FrameCounterUp = uint16(fc)
		case "fcntdn":
			fc, err := strconv.Atoi(keyval[1])
			if err != nil {
				fmt.Println("Invalid value for fcntdn: ", keyval[1])
				return true
			}
			newDevice.FrameCounterDown = uint16(fc)
		default:
			fmt.Println("Unknown property: ", keyval[1])
			fmt.Println("Known properties: nwkskey, appskey, type, relaxedcounter, appkey, devaddr, eui, fcntup, fcntdn")
			return true
		}
	}
	dev, err := n.congress.CreateDevice(n.appEUI, newDevice)
	if err != nil {
		fmt.Println("Error creating device: ", err)
		return true
	}

	keys, _ := NewDeviceKeys(n.mode.AppEUI, dev)
	e := NewEmulatedDevice(n.mode.Config, keys, n.mode.OutgoingMessages, n.mode.Publisher)
	id := interactiveDevice{device: dev, emulated: e}
	n.mode.devices = append(n.mode.devices, id)
	fmt.Println("Device created")
	return true
}

func (n *newDevice) Help() string {
	return helpText("n, new", "create device", "<otaa/abp> <key>=<value>")
}

// -------------------------------------------------
// remove device
type removeDevice struct {
	appEUI   string
	congress *lassie.Client
	mode     *InteractiveMode
}

func (r *removeDevice) Matches(cmd string) bool {
	return (strings.HasPrefix(cmd, "d"))
}

func (r *removeDevice) Help() string {
	return helpText("d, del", "remove device", "")
}

func (r *removeDevice) Execute(params []string) bool {

	if len(r.mode.devices) == 0 {
		fmt.Println("No devices")
		return true
	}

	deviceToRemove := r.mode.devices[r.mode.selectedDevice]
	if err := r.congress.DeleteDevice(r.appEUI, deviceToRemove.device.EUI); err != nil {
		fmt.Println("Error removing device: ", err)
		return true
	}

	r.mode.devices = append(r.mode.devices[0:r.mode.selectedDevice], r.mode.devices[r.mode.selectedDevice+1:]...)
	fmt.Println("Device removed")
	r.mode.selectedDevice = 0
	return true
}

// -------------------------------------------------
// select device
type selectDevice struct {
	mode *InteractiveMode
}

func (s *selectDevice) Matches(cmd string) bool {
	return (strings.HasPrefix(cmd, "s"))
}

func (s *selectDevice) Help() string {
	return helpText("s, sel", "select device", "<ID>")
}

func (s *selectDevice) Execute(params []string) bool {
	if len(params) == 0 {
		fmt.Println("Needs device ID")
		return true
	}
	id, err := strconv.Atoi(params[0])
	if err != nil || id < 0 || id > (len(s.mode.devices)-1) {
		fmt.Println("Invalid device ID: ", params[0])
		return true
	}
	s.mode.selectedDevice = id
	fmt.Println("Selected device is now #", id)
	return true
}

// -------------------------------------------------
// join device
type joinDevice struct {
	mode *InteractiveMode
}

func (j *joinDevice) Matches(cmd string) bool {
	return strings.HasPrefix(cmd, "j")
}

func (j *joinDevice) Help() string {
	return helpText("j, join", "start OTAA join", "")
}

func (j *joinDevice) Execute(params []string) bool {
	if len(j.mode.devices) == 0 {
		fmt.Println("No device.")
		return true
	}
	idevice := j.mode.devices[j.mode.selectedDevice]
	if idevice.device.Type == "ABP" {
		fmt.Println("Only OTAA devices can join")
		return true
	}

	if err := idevice.emulated.Join(joinAttempts); err != nil {
		fmt.Println("Device couldn't join network: ", err)
	}
	return true
}

// -------------------------------------------------
// send message
type sendMessage struct {
	mode *InteractiveMode
}

func (s *sendMessage) Matches(cmd string) bool {
	return strings.HasPrefix(cmd, "s") || strings.HasPrefix(cmd, "m")
}

func (s *sendMessage) Help() string {
	return helpText("m, msg", "send message", "<hex> <ack>")
}

func (s *sendMessage) Execute(params []string) bool {
	idevice := s.mode.devices[s.mode.selectedDevice]
	mtype := protocol.UnconfirmedDataUp
	var payload []byte
	for _, v := range params {
		if v == "ack" || v == "confirmed" || v == "true" {
			mtype = protocol.ConfirmedDataUp
		} else {
			// Assume hex payload
			buf, err := hex.DecodeString(v)
			if err != nil {
				fmt.Println("Invalid payload: ", v)
				return true
			}
			payload = buf
		}
	}
	if err := idevice.emulated.SendMessageWithPayload(mtype, payload); err != nil {
		fmt.Printf("Send error: %v\n", err)
		return true
	}
	fmt.Printf("Message sent successfully.\n")
	return true
}

// -------------------------------------------------
// print device
type printDevice struct {
	mode *InteractiveMode
}

func (p *printDevice) Matches(cmd string) bool {
	return strings.HasPrefix(cmd, "p")
}

func (p *printDevice) Help() string {
	return helpText("p, print", "print device details", "")
}

func (p *printDevice) Execute(params []string) bool {
	if (len(p.mode.devices) - 1) < p.mode.selectedDevice {
		fmt.Println("No devices")
		return true
	}

	i := p.mode.devices[p.mode.selectedDevice]

	fmt.Println()
	fmt.Println("Settings ---------------------------------------")
	fmt.Println("EUI            : ", i.device.EUI)
	fmt.Println("AppKey         : ", i.device.ApplicationKey)
	fmt.Println("AppSKey        : ", i.device.ApplicationSessionKey)
	fmt.Println("NwkSKey        : ", i.device.NetworkSessionKey)
	fmt.Println("RelaxedCounter : ", i.device.RelaxedCounter)
	fmt.Println("FCntUp         : ", i.emulated.FrameCounterUp)
	fmt.Println("FCntDn         : ", i.emulated.FrameCounterDown)
	fmt.Println()
	fmt.Println("Messages ----------------------------------------")
	for _, v := range i.emulated.ReceivedMessages {
		fmt.Printf("%10s : %s\n", v.MessageType, v.Payload)
	}
	fmt.Println()
	return true
}

// -------------------------------------------------
// print device
type ackMessage struct {
	mode *InteractiveMode
}

func (a *ackMessage) Matches(cmd string) bool {
	return strings.HasPrefix(cmd, "a")
}

func (a *ackMessage) Help() string {
	return helpText("a, ack", "set ack flag for next upstream message", "")
}

func (a *ackMessage) Execute(params []string) bool {
	if len(a.mode.devices) == 0 {
		fmt.Printf("No device.")
		return true
	}
	a.mode.devices[a.mode.selectedDevice].emulated.Ack = true
	fmt.Println("ACK flag for next upstream message set.")
	return true
}
