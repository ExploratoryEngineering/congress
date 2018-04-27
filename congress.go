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
	"errors"

	"github.com/ExploratoryEngineering/congress/gateway"
	"github.com/ExploratoryEngineering/congress/monitoring"
	"github.com/ExploratoryEngineering/congress/processor"
	"github.com/ExploratoryEngineering/congress/restapi"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/congress/storage/dbstore"
	"github.com/ExploratoryEngineering/congress/storage/memstore"
	"github.com/ExploratoryEngineering/logging"
	"github.com/ExploratoryEngineering/pubsub"
)

// Server is the main Congress server process. It will launch several
// endpoints and a processing pipeline.
type Server struct {
	config     *server.Configuration
	context    *server.Context
	forwarder  processor.GwForwarder
	pipeline   *processor.Pipeline
	monitoring *monitoring.Endpoint
	restapi    *restapi.Server
	terminator chan bool
}

func (c *Server) setupLogging() {
	logging.SetLogLevel(c.config.LogLevel)

	if c.config.Syslog {
		logging.EnableSyslog()
		logging.Debug("Using syslog for logs, log level is %d", c.config.LogLevel)
	} else {
		logging.EnableStderr(c.config.PlainLog)
		logging.Debug("Using stderr for logs, log level is %d", c.config.LogLevel)
	}
}

func (c *Server) checkConfig() error {
	if err := c.config.Validate(); err != nil {
		logging.Error("Invalid configuration: %v Exiting", err)
		return errors.New("invalid configuration")
	}
	return nil
}

// NewServer creates a new server with the given configuration. The configuration
// is checked before the server is created, logging is initialized
func NewServer(config *server.Configuration) (*Server, error) {
	c := &Server{config: config, terminator: make(chan bool)}
	c.setupLogging()

	if err := c.checkConfig(); err != nil {
		return nil, err
	}
	logging.Info("This is the Congress server")

	var datastore storage.Storage
	var err error
	if c.config.DBConnectionString != "" {
		logging.Info("Using PostgreSQL as backend storage")
		datastore, err = dbstore.CreateStorage(config.DBConnectionString,
			config.DBMaxConnections, config.DBIdleConnections, config.DBConnLifetime)
		if err != nil {
			logging.Error("Couldn't connect to database: %v", err)
			return nil, err
		}
	} else if config.MemoryDB {
		logging.Warning("Using in-memory database as backend storage")
		datastore = memstore.CreateMemoryStorage(config.MemoryMinLatencyMs, config.MemoryMaxLatencyMs)
	}

	keyGenerator, err := server.NewEUIKeyGenerator(config.RootMA(), uint32(config.NetworkID), datastore.Sequence)
	if err != nil {
		logging.Error("Could not create key generator: %v. Terminating.", err)
		return nil, errors.New("unable to create key generator")
	}
	frameOutput := server.NewFrameOutputBuffer()

	appRouter := pubsub.NewEventRouter(5)
	gwEventRouter := pubsub.NewEventRouter(5)
	c.context = &server.Context{
		Storage:       &datastore,
		Terminator:    make(chan bool),
		FrameOutput:   &frameOutput,
		Config:        config,
		KeyGenerator:  &keyGenerator,
		GwEventRouter: &gwEventRouter,
		AppRouter:     &appRouter,
		AppOutput:     server.NewAppOutputManager(&appRouter),
	}

	logging.Info("Launching generic packet forwarder on port %d...", config.GatewayPort)
	c.forwarder = gateway.NewGenericPacketForwarder(c.config.GatewayPort, datastore.Gateway, c.context)
	c.pipeline = processor.NewPipeline(c.context, c.forwarder)
	c.restapi, err = restapi.NewServer(config.OnlyLoopback, c.context, c.config)
	if err != nil {
		logging.Error("Unable to create REST API endpoint: %v", err)
		return nil, err
	}

	c.monitoring, err = monitoring.NewEndpoint(true, config.DebugPort, config.ProfilingEndpoint, config.RuntimeTrace)
	if config.ProfilingEndpoint {
		logging.Warning("Profiling is turned ON - access monitoring endpoint to inspect")
	}
	if err != nil {
		logging.Error("Unable to create monitoring endpoint: %v", err)
		return nil, err
	}
	return c, nil
}

// Start Starts the congress server
func (c *Server) Start() error {
	logging.Debug("Starting pipeline")
	c.pipeline.Start()
	logging.Debug("Starting forwarder")
	go c.forwarder.Start()

	if err := c.monitoring.Start(); err != nil {
		logging.Error("Unable to launch monitoring endpoint: %v", err)
		return err
	}
	logging.Warning("Monitoring is available at http://localhost:%d/debug", c.monitoring.Port())

	logging.Debug("Launching outputs")
	go c.context.AppOutput.LoadOutputs(c.context.Storage.AppOutput)

	logging.Debug("Launching http server")
	if err := c.restapi.Start(); err != nil {
		logging.Error("Unable to start REST API endpoint: %v", err)
		return err
	}
	logging.Info("Server is ready and serving HTTP on port %d", c.config.HTTPServerPort)

	return nil
}

// Shutdown stops the Congress server.
func (c *Server) Shutdown() error {
	c.forwarder.Stop()
	c.restapi.Shutdown()
	c.monitoring.Shutdown()
	c.context.Storage.Close()

	return nil
}
