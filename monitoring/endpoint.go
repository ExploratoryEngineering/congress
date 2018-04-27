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
	"errors"
	"expvar"
	"fmt"
	"os"
	"path"
	"time"

	"context"
	"net/http"
	"net/http/pprof"

	"github.com/ExploratoryEngineering/congress/utils"
	"github.com/ExploratoryEngineering/logging"
)

// This is the default endpoint for the expvar package. The location isn't
// ideal (/varz) but this is the default for Go so it makes sense to use it as
// is.
const defaultEndpoint = "/debug/vars"

// Endpoint is a type that can launch a http monitoring endpoint.
type Endpoint struct {
	srv  *http.Server
	port int
	mux  *http.ServeMux
	bind string
}

// NewEndpoint returns a new Endpoint instance
func NewEndpoint(loopbackOnly bool, port int, profiling bool, tracing bool) (*Endpoint, error) {
	ret := &Endpoint{}
	portno := port
	var err error
	if portno == 0 {
		portno, err = utils.FreePort()
		if err != nil {
			return nil, err
		}
	}
	ret.port = portno

	host := ""
	if loopbackOnly {
		host = "localhost"
	}
	ret.mux = http.NewServeMux()
	ret.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is the monitoring endpoint"))
	})
	ret.mux.Handle(defaultEndpoint, expvar.Handler())
	if profiling {
		ret.mux.HandleFunc("/debug/pprof/", pprof.Index)
		ret.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		ret.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		ret.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}
	if tracing {
		EnableTracing()
		ret.mux.HandleFunc("/debug/trace", TraceHandler())
	}
	execpath, err := os.Executable()
	if err == nil {
		uipath := path.Join(path.Dir(execpath), "ui")
		if _, err := os.Stat(path.Join(uipath, "index.html")); err == nil {
			logging.Info("Serving debug UI at /debug/ui")
			ret.mux.Handle("/debug/ui/", http.StripPrefix("/debug/ui/", http.FileServer(http.Dir(uipath))))
		}
	}
	ret.srv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, ret.port),
		Handler: ret,
	}

	return ret, nil
}

// Start launches the server. If the port is set to 0 it will pick a random
// port to run on.
func (m *Endpoint) Start() error {
	if m.srv == nil {
		return errors.New("No valid server")
	}
	go func() {
		if err := m.srv.ListenAndServe(); err != http.ErrServerClosed {
			logging.Error("Unable to listen and serve: %v", err)
		}
	}()
	return nil
}

func (m *Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mux.ServeHTTP(w, r)
}

// Shutdown stops the server. There is a 2 second timeout.
func (m *Endpoint) Shutdown() error {
	if m.srv == nil {
		return errors.New("server not launched yet")
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelFunc()
	if err := m.srv.Shutdown(ctx); err != nil {
		return err
	}
	m.port = 0
	m.srv = nil
	return nil
}

// Port returns the port the server is running on
func (m *Endpoint) Port() int {
	return m.port
}
