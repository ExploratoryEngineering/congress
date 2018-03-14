# Congress - The LoRa backend server

## Features

* Supports LoRaWAN 1.1
* PostgreSQL or in-memory database
* OAuth integration with Telenor CONNECT
* Token-based API access
* Monitoring via separate endpoint
* REST API to manage applications, gateways and devices
* Websocket, MQTT, AWS IOT outputs
* Gateway interface with the reference packet forwarder from Semtech.

## Not-features

Not-features (ie features not implemented yet)

* No support for frequencies outside EU868. Additional frequencies are
  relatively simple to implement
* No ADR support (yet)
* Limited frequency management
* Custom frequency plans for gateways.
* Redundancy for instances. There's only one server and if you plan to run
  this in a production environment it is highly recommended to implement some
  sort of failover, either by using Nginx or through another kind of load
  balancer.

## Deploying Congress

You can run Congress either as a standalone server locally, in Docker
containers or in AWS.

### Running a standalone process

There are several command line options. List them with `./congress -help`. In
short: If you want to launch a simple server lauch it with
`./congress --disable-auth`. This will bring up a server with no authentication,
logging to stderr, memory-backed storage and a minimum configuration. If you
want to persist data between launches use a PostgreSQL database. Get the script
by running `./congress -printschema`.

### Running via Docker

A Docker-compose configuration file can be found in `deployment/docker`. More
details can be found in the [README](deployment/docker/README.md).

## Testing tools

We've made a few tools that can be used when testing Congress:

* Eagle One - emulates a packet forwarder that can run in both interactive and
  batch mode. The batch mode creates a fixed number of devices, sends messages
  and removes the devices afterwards. In interactive mode you can create
  individual devices and send specific messages.
* Datagenerator - generates data for your backend. The tool uses a direct
  PostgreSQL database connection to create data.

## Let's Encrypt certificates

The server supports Let's Encrypt certificates out of the box. Configure the
certificates with the parameters `acme-cert`, `acme-hostname` and `acme-secret-dir`.

Note that the hostname must be reachable from the internet, ie the server has
to be running on its own domain.

The `acme-secret-dir` is the directory where the private key for the certificate
will be cached.