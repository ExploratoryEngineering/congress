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
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage/dbstore"
	"github.com/ExploratoryEngineering/logging"
)

var config = server.NewDefaultConfig()

func init() {
	flag.IntVar(&config.GatewayPort, "gwport", server.DefaultGatewayPort, "Port for gateway listener")
	flag.IntVar(&config.HTTPServerPort, "http", server.DefaultHTTPPort, "HTTP port to listen on")
	flag.UintVar(&config.NetworkID, "netid", server.DefaultNetworkID, "The Network ID to use")
	flag.StringVar(&config.MA, "ma", server.DefaultMA, "MA to use when generating new EUIs")
	flag.StringVar(&config.DBConnectionString, "connectionstring", "", "Database connection string")
	flag.BoolVar(&config.PrintSchema, "printschema", false, "Print schema definition")
	flag.BoolVar(&config.Syslog, "syslog", false, "Send logs to syslog")
	flag.BoolVar(&config.DisableGatewayChecks, "disablegwcheck", false, "Disable ALL gateway checks")
	flag.StringVar(&config.ConnectHost, "connect-host", server.DefaultConnectHost, "CONNECT ID host")
	flag.StringVar(&config.ConnectClientID, "connect-clientid", server.DefaultConnectClientID, "CONNECT ID client ID")
	flag.StringVar(&config.ConnectRedirectLogin, "connect-login-redirect", "", "CONNECT ID redirect URI for login")
	flag.StringVar(&config.ConnectRedirectLogout, "connect-logout-redirect", "", "CONNECT ID redirect URI for logout")
	flag.StringVar(&config.ConnectPassword, "connect-password", "", "CONNECT ID password")
	flag.BoolVar(&config.DisableAuth, "disable-auth", false, "Disable the authentication layers")
	flag.StringVar(&config.ConnectLoginTarget, "connect-login-target", "", "Final redirect after login roundtrip (internal)")
	flag.StringVar(&config.ConnectLogoutTarget, "connect-logout-target", "", "Final redirect after logout roundtrip (internal)")
	flag.StringVar(&config.TLSCertFile, "tls-cert", "", "TLS certificate")
	flag.StringVar(&config.TLSKeyFile, "tls-key", "", "TLS key file")
	flag.BoolVar(&config.UseSecureCookie, "securecookie", false, "Set the secure flag for the auth cookie")
	flag.UintVar(&config.LogLevel, "loglevel", server.DefaultLogLevel, "Log level to use (0 = debug, 1 = info, 2 = warning, 3 = error)")
	flag.BoolVar(&config.PlainLog, "plainlog", false, "Use plain-text stderr logs")
	flag.BoolVar(&config.MemoryDB, "memorydb", true, "Use in-memory database for storage (for testing)")
	flag.BoolVar(&config.ProfilingEndpoint, "pprof", false, "Turn on profiling endpoint (in monitoring; /debug/pprof/profile)")
	flag.BoolVar(&config.RuntimeTrace, "trace", false, "Turn on runtime trace generation. For testing")
	flag.IntVar(&config.MemoryMinLatencyMs, "min-memdb-latency", 0, "Minimum emulated latency for memory storage")
	flag.IntVar(&config.MemoryMaxLatencyMs, "max-memdb-latency", 0, "Maximum emulated latency for memory storage")
	flag.IntVar(&config.DBMaxConnections, "db-max-connections", server.DefaultMaxConns, "Maximum DB connections")
	flag.IntVar(&config.DBIdleConnections, "db-max-idle-connections", server.DefaultIdleConns, "Maximum idle DB connections")
	flag.DurationVar(&config.DBConnLifetime, "db-max-lifetime-connections", server.DefaultConnLifetime, "Maximum life time of DB connections")
	flag.BoolVar(&config.ACMECert, "acme-cert", false, "Enable Let's Encrypt certificates. Requires host name")
	flag.StringVar(&config.ACMEHost, "acme-hostname", "", "Host name to use when requesting certificates from Let's Encrypt")
	flag.StringVar(&config.ACMESecretDir, "acme-secret-dir", "secret-dir", "Directory for ACME certificate secrets")
	flag.Parse()
}

func main() {
	if config.PrintSchema {
		fmt.Println(dbstore.DBSchema)
		return
	}
	logging.SetLogLevel(config.LogLevel)
	congress, err := NewServer(config)
	if err != nil {
		return
	}

	terminator := make(chan bool)

	if err := congress.Start(); err != nil {
		logging.Error("Congress did not start: %v", err)
		return
	}
	defer func() {
		logging.Info("Congress is shutting down...")
		congress.Shutdown()
		logging.Info("Congress has shut down")
	}()

	sigch := make(chan os.Signal, 2)
	signal.Notify(sigch, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigch
		logging.Debug("Caught signal '%v'", sig)
		terminator <- true
	}()

	<-terminator

}
