package restapi

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
import "strings"

type templateParameters struct {
	OTAA      bool
	DeviceEUI string // Formattted as xx-xx-xx...
	AppEUI    string // Formattted as xx-xx-xx...
	AppKey    string // Formatted as a series of (hex) bytes
	AppSKey   string // Formatted as a series of (hex) bytes
	NwkSKey   string // Formatted as a series of (hex) bytes
	DevAddr   string // Formatted as 0xAABBCCDD
}

// getSourceTemplate returns a tempplate for a particular source
func getSourceTemplate(kind string) string {
	switch strings.ToLower(kind) {
	case "lopy":
		return `
from network import LoRa
import socket
import time
import binascii
import pycom
import  machine

def flash_led(col):
	pycom.rgbled(col)
	time.sleep(0.1)
	pycom.rgbled(0x000000)
	time.sleep(0.1)

# Turn off the heartbeat blink and the LED
pycom.heartbeat(False)
pycom.rgbled(0x000000)

# Initialize LoRa in LORAWAN mode.
lora = LoRa(mode=LoRa.LORAWAN)

# Device provisioning. Use either OTAA or ABP
OTAA_DEVICE = {{ if .OTAA }} True {{ else }} False {{ end }}

# OTAA parameters
dev_eui = bytes([ {{ .DeviceEUI }}])
app_eui = bytes([ {{ .AppEUI }}])
app_key = bytes([ {{ .AppKey }}])

# ABP parameters
dev_addr = {{ .DevAddr }}
apps_key = bytes([ {{ .AppSKey }} ])
nwks_key = bytes([ {{ .NwkSKey }} ])

if OTAA_DEVICE:
	lora.join(activation=LoRa.OTAA, auth=(dev_eui, app_eui, app_key), timeout=0)
else:
	lora.join(activation=LoRa.ABP, auth=(dev_addr, nwks_key, apps_key), timeout=0)

# wait until the module has joined the network
while not lora.has_joined():
	flash_led(0xff0000)

for i in range(1, 2):
	flash_led(0x0000ff)

s = socket.socket(socket.AF_LORA, socket.SOCK_RAW)

# set the LoRaWAN data rate
s.setsockopt(socket.SOL_LORA, socket.SO_DR, 5)

# create a raw LoRa socket
s = socket.socket(socket.AF_LORA, socket.SOCK_RAW)

while True:

	flash_led(0xffff00)
	s.setblocking(True)
	s.send(bytes([0x11, 0x22, 0x33, 0x44, 0x55]))
	flash_led(0x00ffff)

	s.setblocking(False)
	data = s.recv(64)
	if len(data) > 0:
		print("Received data:",  data)

	time.sleep(10)
`
	case "mynewt":
		fallthrough
	case "lmic":
		fallthrough
	case "c":
		fallthrough
	default:
		return `
#define LORAWAN_OTAA {{ if .OTAA }} 1 {{ else }} 0 {{ end }}

/* ==========================================================================
	* OTAA provisioning. This includes the device EUI, application EUI and
	* application key. The application and network session keys will be
	* negotiated when the device joins the network.
	* ========================================================================== */
#define LORAWAN_DEVICE_EUI { {{ .DeviceEUI }} }
#define LORAWAN_APP_KEY { {{ .AppKey }} }
#define LORAWAN_APP_EUI { {{ .AppEUI }} }

/* ==========================================================================
	* ABP (activation by personalisation) parameters. Rather than negotiating
	* the application and network session keys these are set directly. This
	* eliminates the need for a join request and the device can transmit data
	* directly. The downside is that the device should keep track of the frame
	* counters when it is powered down but you won't share the application key.
	* ========================================================================== */
#define LORAWAN_DEVICE_ADDRESS (uint32_t) {{ .DevAddr }}
#define LORAWAN_NWKSKEY { {{ .NwkSKey }} }
#define LORAWAN_APPSKEY { {{ .AppSKey }} }
`
	}
}
