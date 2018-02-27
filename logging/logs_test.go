package logging
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

var levels = []uint{
	DebugLevel,
	InfoLevel,
	WarningLevel,
	ErrorLevel,
}

func TestStderrLogging(t *testing.T) {
	EnableStderr(true)

	for i, v := range levels {
		SetLogLevel(v)
		Debug("This is debug level (round %d)", i)
		Info("This is info level (round %d)", i)
		Warning("This is warning level (round %d)", i)
		Error("This is error level (round %d)", i)
	}

	EnableStderr(false)
	for i, v := range levels {
		SetLogLevel(v)
		Debug("This is debug level (round %d)", i)
		Info("This is info level (round %d)", i)
		Warning("This is warning level (round %d)", i)
		Error("This is error level (round %d)", i)
	}
}

func TestSyslogLogging(t *testing.T) {
	EnableSyslog()

	for i, v := range levels {
		SetLogLevel(v)
		Debug("This is debug level (round %d)", i)
		Info("This is info level (round %d)", i)
		Warning("This is warning level (round %d)", i)
		Error("This is error level (round %d)", i)
	}
}
