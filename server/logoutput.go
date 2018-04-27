package server

import "github.com/ExploratoryEngineering/congress/model"

// Testing code below -------------------------------------------------------
// logTransport is a simple log-only transport configuration. This is only
// used for testing.
type logTransport struct {
}

func newLogTransport(tc model.TransportConfig) transport {
	return &logTransport{}
}

func (d *logTransport) open(ml *MemoryLogger) bool {
	return true
}

func (d *logTransport) send(msg interface{}, ml *MemoryLogger) bool {
	return true
}
func (d *logTransport) close(ml *MemoryLogger) {
}

func init() {
	transports["log"] = newLogTransport
}
