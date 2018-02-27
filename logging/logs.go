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
import (
	"fmt"
	"log"
	"log/syslog"
	"os"
)

// Debug is the log file for debug logging. It is disabled by default. These
// messages aren't really useful for anything except the developers.
var debug = log.New(os.Stderr, " ", stderrFlags)

// Info is the log for info-level logs. It is disabled by default. These messages
// are typically "application created", "user registered", "device deleted" and
// so on. They are useful when doing detailed monitoring of the service
var info = log.New(os.Stderr, "  ", stderrFlags)

// Warning is the log for warning-level logs, typically inconsistencies that
// users might notice. There are *some* messages in this but not a lot.
var warning = log.New(os.Stderr, " ", stderrFlags)

// Error is the log for severe error logs; database errors, data inconsistencies,
// failures and issues that require immediate action. There are very few issues
// on this scale.
var errlog = log.New(os.Stderr, " ", stderrFlags)

// LogLevel is the log detail level

const (
	// DebugLevel is the most detailed logging level. It will emit all log levels.
	DebugLevel uint = iota
	// InfoLevel is the log level that will log info, warning and errors
	InfoLevel
	// WarningLevel is the log level that will log warnings and errors
	WarningLevel
	// ErrorLevel is the log level that only logs errors
	ErrorLevel
)

var currentLevel uint

const syslogFlags = log.Lshortfile
const stderrFlags = log.Ldate + log.Ltime + log.Lshortfile

func init() {
	EnableStderr(true)
	SetLogLevel(WarningLevel)
}

// SetLogLevel sets the logging level
func SetLogLevel(level uint) {
	currentLevel = level
}

func setFlags(flags int) {
	debug.SetFlags(flags)
	info.SetFlags(flags)
	warning.SetFlags(flags)
	errlog.SetFlags(flags)
}

// EnableSyslog enables syslog logging.
func EnableSyslog() {
	errorLog, err := syslog.New(syslog.LOG_ERR|syslog.LOG_DAEMON, "congress")
	if err != nil {
		errlog.Printf("Unable to set up error syslog: %v", err)
		return
	}
	warningLog, err := syslog.New(syslog.LOG_WARNING|syslog.LOG_DAEMON, "congress")
	if err != nil {
		errlog.Printf("Unable to set up warning syslog: %v", err)
		return
	}
	infoLog, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "congress")
	if err != nil {
		errlog.Printf("Unable to set up info syslog: %v", err)
		return
	}
	debugLog, err := syslog.New(syslog.LOG_DEBUG|syslog.LOG_DAEMON, "congress")
	if err != nil {
		errlog.Printf("Unable to set up debug syslog: %v", err)
		return
	}
	debug.SetOutput(debugLog)
	info.SetOutput(infoLog)
	warning.SetOutput(warningLog)
	errlog.SetOutput(errorLog)
	// Syslog includes time stamp so we just need the source file
	setFlags(syslogFlags)

	// Set text prefixes since that makes it easier to search the syslog
	debug.SetPrefix("")
	info.SetPrefix("")
	warning.SetPrefix("")
	errlog.SetPrefix("")
	SetLogLevel(currentLevel)
}

// ANSI escape codes for colored log lines. This will only show up on the stderr
// logs. Printf statements on stdout will be broken but we don't do printf's do
// we?
const (
	debugText   = "\x1b[0m"    // White
	infoText    = "\x1b[34;1m" // Bright blue
	warningText = "\x1b[33;1m" // Bright yellow
	errorText   = "\x1b[31;1m" // Bright red
)

// EnableStderr enables logging to stderr
func EnableStderr(plainText bool) {
	logwriter := os.Stderr

	debug.SetOutput(logwriter)
	info.SetOutput(logwriter)
	warning.SetOutput(logwriter)
	errlog.SetOutput(logwriter)

	setFlags(stderrFlags)

	if plainText {
		// Use plain text logging
		debug.SetPrefix("DEBUG   ")
		info.SetPrefix("INFO    ")
		warning.SetPrefix("WARNING ")
		errlog.SetPrefix("ERROR   ")
	} else {
		// Use fancy emojis as prefix since this is something we'll look a *lot* at.
		debug.SetPrefix(debugText + "    ")
		info.SetPrefix(infoText + "ℹ️   ")
		warning.SetPrefix(warningText + "⚠️   ")
		errlog.SetPrefix(errorText + "🛑   ")
	}

	SetLogLevel(currentLevel)
}

// Debug adds a debug-level log message to the log. If the log level is set
// higher than DebugLevel the message will be discarded.
func Debug(format string, v ...interface{}) {
	if currentLevel == DebugLevel {

		debug.Output(2, fmt.Sprintf(format, v...))
	}
}

// Info adds an info-level log message to the log if the log level is set
// to InfoLevel or lower.
func Info(format string, v ...interface{}) {
	if currentLevel <= InfoLevel {
		info.Output(2, fmt.Sprintf(format, v...))
	}
}

// Warning adds a warning-level log message if the log level is set to
// WarningLevel or lower.
func Warning(format string, v ...interface{}) {
	if currentLevel <= WarningLevel {
		warning.Output(2, fmt.Sprintf(format, v...))
	}
}

// Error adds an error-level log message to the log.
func Error(format string, v ...interface{}) {
	errlog.Output(2, fmt.Sprintf(format, v...))
}
