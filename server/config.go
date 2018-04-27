package server

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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ExploratoryEngineering/congress/protocol"
	"github.com/ExploratoryEngineering/logging"
)

// Configuration holds the configuration for the system
type Configuration struct {
	GatewayPort           int
	HTTPServerPort        int
	NetworkID             uint   // The network ID that this instance handles. The default is 0
	MA                    string // String representation of MA
	DBConnectionString    string
	PrintSchema           bool
	Syslog                bool
	DisableGatewayChecks  bool
	ConnectHost           string
	ConnectClientID       string
	ConnectRedirectLogin  string
	ConnectRedirectLogout string
	ConnectPassword       string
	DisableAuth           bool
	ConnectLoginTarget    string
	ConnectLogoutTarget   string
	TLSCertFile           string
	TLSKeyFile            string
	UseSecureCookie       bool
	LogLevel              uint
	PlainLog              bool // Fancy stderr logs with emojis and colors
	MemoryDB              bool
	OnlyLoopback          bool // use only loopback adapter - for testing
	ProfilingEndpoint     bool // Turn on profiling endpoint - for testing
	RuntimeTrace          bool // Turn on runtime trace - for testing
	MemoryMinLatencyMs    int
	MemoryMaxLatencyMs    int
	DebugPort             int // Debug port - 0 for random, default 8081
	DBMaxConnections      int
	DBIdleConnections     int
	DBConnLifetime        time.Duration
	ACMECert              bool   // AutoCert via Let's Encrypt
	ACMEHost              string // AutoCert hostname
	ACMESecretDir         string
}

// This is the default configuration
const (
	DefaultGatewayPort     = 8000
	DefaultHTTPPort        = 8080
	DefaultDebugPort       = 8081
	DefaultNetworkID       = 0
	DefaultMA              = "00-09-09"
	DefaultConnectHost     = "connect.staging.telenordigital.com"
	DefaultConnectClientID = "telenordigital-connectexample-web"
	DefaultLogLevel        = 0
	DefaultMaxConns        = 200
	DefaultIdleConns       = 100
	DefaultConnLifetime    = 10 * time.Minute
)

// NewDefaultConfig returns the default configuration. Note that this configuration
// isn't valid right out of the box; a storage backend must be selected.
func NewDefaultConfig() *Configuration {
	return &Configuration{
		MA:                DefaultMA,
		HTTPServerPort:    DefaultHTTPPort,
		NetworkID:         DefaultNetworkID,
		ConnectClientID:   DefaultConnectClientID,
		ConnectHost:       DefaultConnectHost,
		LogLevel:          DefaultLogLevel,
		DebugPort:         DefaultDebugPort,
		DBMaxConnections:  DefaultMaxConns,
		DBConnLifetime:    DefaultConnLifetime,
		DBIdleConnections: DefaultIdleConns,
	}
}

// NewMemoryNoAuthConfig returns a configuration with no authentication and
// memory-backed storage. This is a valid configuration.
func NewMemoryNoAuthConfig() *Configuration {
	ret := NewDefaultConfig()
	ret.MemoryDB = true
	ret.DisableAuth = true
	return ret
}

// RootMA returns the MA to use as the base MA for EUIs. The configuration
// is assumed to be valid at this point. If there's an error converting the
// MA it will panic.
func (cfg *Configuration) RootMA() protocol.MA {
	prefix, err := hex.DecodeString(strings.Replace(cfg.MA, "-", "", -1))
	if err != nil {
		panic("invalid format for MA string in configuration")
	}
	ret, err := protocol.NewMA(prefix)
	if err != nil {
		panic("unable to create MA")
	}
	return ret
}

// Validate checks the configuration for inconsistencies and errors. This
// function logs the warnings using the logger package as well.
func (cfg *Configuration) Validate() error {
	prefix, err := hex.DecodeString(strings.Replace(cfg.MA, "-", "", -1))
	if err != nil {
		return fmt.Errorf("invalid format for MA string: %v", err)
	}
	_, err = protocol.NewMA(prefix)
	if err != nil {
		return fmt.Errorf("unable to create MA: %v", err)
	}
	if cfg.ConnectClientID == DefaultConnectClientID {
		logging.Warning("Using the default Connect Client ID (%s). This will only work for servers running locally", cfg.ConnectClientID)
	}
	if cfg.DisableAuth {
		logging.Warning("The authentication layers are DISABLED. The REST API will NOT authenticate!")
	}

	if (cfg.TLSCertFile != "" && cfg.TLSKeyFile == "") || (cfg.TLSCertFile == "" && cfg.TLSKeyFile != "") {
		return fmt.Errorf("both TLS cert file (%s) and TLS key file (%s) must be specified", cfg.TLSCertFile, cfg.TLSKeyFile)
	}

	if cfg.TLSCertFile == "" {
		logging.Warning("Running server without TLS")
	}
	if cfg.DBConnectionString == "" && !cfg.MemoryDB {
		return errors.New("no backend storage selected. A connection string, embedded PostgreSQL or in-memory database must be selected")
	}

	if cfg.MemoryMaxLatencyMs < cfg.MemoryMinLatencyMs {
		return errors.New("min memory storage latency must be less than max memory storage latency")
	}
	if cfg.MemoryMaxLatencyMs > 0 && cfg.MemoryMinLatencyMs == cfg.MemoryMaxLatencyMs {
		return errors.New("min and max memory latency cannot be equal")
	}
	if cfg.ACMECert && cfg.ACMEHost == "" {
		return errors.New("ACME hostname must be set if ACME certs are used")
	}
	return nil
}
