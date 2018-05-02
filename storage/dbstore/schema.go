package dbstore

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

// DBSchema contains the storage scheme for PostgreSQL
const DBSchema = `
-- **************************************************************************
-- This is the canonical storage schema for the LoRa backend.
-- **************************************************************************
-- * Identifiers are EUI-64, ie 16 char hex strings. Hex strings are stored
--   as is without any whitespace. Storing as strings aren't the most efficient
--   way but makes it easy to maintain the database. This can be changed to
--   a binary representation later if required.
-- * Keys are 128 bits, 32 char hex strings. Keys are stored as is without
--   any whitespace.

-- **************************************************************************
-- Users. The user information is cached from the Connect ID servers. Emails
-- are only added if they're verified. We're not using this information ATM
-- and will rely on Connect ID for up to date information but this comes in
-- handy if we have to notify the users via email.
-- **************************************************************************
CREATE TABLE lora_user (
    user_id  VARCHAR(128)   NOT NULL, -- the user ID, connect ID ATM
    name     VARCHAR(128)   NULL,     -- (optional) user name
    email    VARCHAR(128)   NULL,     -- (optional) verified email

    CONSTRAINT lora_user_pk PRIMARY KEY (user_id)
);
CREATE INDEX lora_user_name ON lora_user (name);
CREATE INDEX lora_user_email ON lora_user (email);

-- **************************************************************************
-- API tokens. Tokens are identified by their token; each token controls
-- read or write access to a specified resource.
-- **************************************************************************
CREATE TABLE lora_token (
    token     VARCHAR(64)   NOT NULL,
    resource  VARCHAR(128)  NOT NULL,
    user_id   VARCHAR(128)  NOT NULL REFERENCES lora_user (user_id) ON DELETE CASCADE,
    write     BOOL          NOT NULL DEFAULT FALSE,
    tags      JSONB         NULL,

    CONSTRAINT lora_token_pk PRIMARY KEY (token)
);
CREATE INDEX lora_token_user ON lora_token(user_id);

-- **************************************************************************
-- Organization. An organization is just a collection of users. Users that are
-- member of the organization will be able to access networks, gateways and
-- applications that the organization owns.
-- **************************************************************************
CREATE TABLE lora_organization (
    org_id BIGINT NOT NULL, -- EUI64 in decimal form
    name VARCHAR(128) NOT NULL,

    CONSTRAINT lora_organization_pk PRIMARY KEY (org_id)
);

-- **************************************************************************
-- Owners of applications, gateways and networks. The owners can be either
-- users or organizations; this table links to the relevant owner. Either the
-- user_id field is set or the org_id field is set.
-- **************************************************************************
CREATE TABLE lora_owner (
    owner_id   BIGINT         NOT NULL, -- EUI64 in decimal form
    user_id    VARCHAR(128)   NULL REFERENCES lora_user ON DELETE CASCADE,
    org_id     BIGINT         NULL REFERENCES lora_organization ON DELETE CASCADE,

    CONSTRAINT lora_owner_pk PRIMARY KEY (owner_id)
);

CREATE INDEX lora_owner_user_id ON lora_owner (user_id NULLS FIRST);
CREATE INDEX lora_owner_org_id ON lora_owner (org_id NULLS FIRST);

-- **************************************************************************
-- Members of organizations. Each member have a role - either read-only or
-- read/write. Read/write equates to admin rights. Owners with read/write
-- permissions can edit the member list for the organization (including the
-- org name). We *might* extend this with admin rights if we want more
-- granularity later on.
-- **************************************************************************
CREATE TABLE lora_org_member (
    org_id   BIGINT        NOT NULL REFERENCES lora_organization (org_id) ON DELETE CASCADE,
    user_id  VARCHAR(128)  NOT NULL REFERENCES lora_user (user_id) ON DELETE CASCADE,
    role     CHAR(1)       NOT NULL, -- 'R' - read only, 'W' - read/write

    CONSTRAINT lora_org_member_pk PRIMARY KEY (org_id, user_id)
);

-- **************************************************************************
-- Applications. There will be thousands if not millions of these. They
-- change rarely.
-- **************************************************************************
CREATE TABLE lora_application (
    eui         CHAR(23)     NOT NULL,
    owner_id    BIGINT       NOT NULL REFERENCES lora_owner (owner_id),
    tags        JSONB        NULL,

    CONSTRAINT lora_application_pk PRIMARY KEY (eui)
);


-- **************************************************************************
-- The devices themselves. There will be millions of devices. They change
-- sometimes but not a lot.
-- **************************************************************************
CREATE TABLE lora_device (
    eui             CHAR(23)  NOT NULL,
    dev_addr        CHAR(8)   NOT NULL,
    app_key         CHAR(32)  NOT NULL,
    apps_key        CHAR(32)  NOT NULL,
    nwks_key        CHAR(32)  NOT NULL,
    application_eui CHAR(23)  NOT NULL REFERENCES lora_application(eui),
    state           SMALLINT  NOT NULL,
    fcnt_up         INTEGER   NOT NULL DEFAULT 0,
    fcnt_dn         INTEGER   NOT NULL DEFAULT 0,
    relaxed_counter BOOLEAN   NOT NULL DEFAULT false,
    key_warning     BOOLEAN   NOT NULL DEFAULT false,
    tags            JSONB     NULL,

    CONSTRAINT lora_device_pk PRIMARY KEY (eui)
);

CREATE INDEX lora_device_application_eui ON lora_device(application_eui);
CREATE INDEX lora_device_dev_addr ON lora_device(dev_addr);
CREATE INDEX lora_device_state ON lora_device(state);

-- **************************************************************************
-- Nonces received from device. Nonces are kept for a while to ensure there
-- are no replay attacks. All devices uses this technique, even ABP devices.
-- **************************************************************************
CREATE TABLE lora_device_nonce (
    device_eui CHAR(23) NOT NULL REFERENCES lora_device (eui) ON DELETE CASCADE,
    nonce      INT  NOT NULL,

    CONSTRAINT lora_device_nonce_pk PRIMARY KEY(device_eui, nonce)
);

-- **************************************************************************
-- Data from device. There might be a *lot* of data from the device so this
-- should be stored in another kind of backend (Cassandra or equivalen) when
-- there's millions and millions of devices. Data will typically be written
-- once and read many times.
-- **************************************************************************
CREATE TABLE lora_device_data (
    device_eui  CHAR(23)      NOT NULL REFERENCES lora_device (eui) ON DELETE CASCADE, -- device address
    data        VARCHAR(512)  NOT NULL, -- base64 encoded payload (unencrypted)
    time_stamp  BIGINT        NOT NULL, -- time stamp
    gateway_eui CHAR(23)      NOT NULL, -- gateway EUI (if available)
    rssi        INTEGER       NOT NULL,
    snr         NUMERIC(6,3)  NOT NULL,
    frequency   NUMERIC(6,3)  NOT NULL,
    data_rate   VARCHAR(20)   NOT NULL,
    dev_addr    CHAR(8)       NOT NULL,

    CONSTRAINT lora_device_data_pk PRIMARY KEY(device_eui, time_stamp)
);

CREATE INDEX lora_device_data_device_eui ON lora_device_data(device_eui);


-- **************************************************************************
-- Sequences. Each sequence is identified by a string and the counter is
-- incremented independently for each sequence.
-- **************************************************************************
CREATE TABLE lora_sequence (
    identifier VARCHAR(128) NOT NULL, -- identifier
    counter    BIGINT       NOT NULL, -- The current counter value

    CONSTRAINT lora_sequence_pk PRIMARY KEY (identifier)
);

CREATE INDEX lora_sequence_identifier ON lora_sequence(identifier);

-- **************************************************************************
-- Gateways. The gateways are fairly self explanatory.
-- **************************************************************************
CREATE TABLE lora_gateway (
    gateway_eui CHAR(23)      NOT NULL,
    latitude    NUMERIC(12,8) NULL,
    longitude   NUMERIC(12,8) NULL,
    altitude    NUMERIC(8,3)  NULL,
    ip          VARCHAR(64)   NOT NULL,
    strict_ip   BOOL          NOT NULL,
    owner_id    BIGINT        NOT NULL REFERENCES lora_owner (owner_id),
    tags        JSONB         NULL,

    CONSTRAINT lora_gateway_pk PRIMARY KEY (gateway_eui)
);

-- Set up initial system user.
INSERT INTO lora_user (user_id, name, email) VALUES ('system', 'System user', 'ee@telenordigital.com');
INSERT INTO lora_owner (owner_id, user_id, org_id) VALUES (0, 'system', null);

-- **************************************************************************
-- Outputs for applications
-- **************************************************************************
CREATE TABLE lora_output (
    eui CHAR(23) NOT NULL,
    config JSONB NULL,
    application_eui CHAR(23) NOT NULL REFERENCES lora_application(eui) ON DELETE CASCADE,

    CONSTRAINT lora_output_pk PRIMARY KEY (eui)
);

CREATE INDEX lora_output_app_eui ON lora_output(application_eui);

-- **************************************************************************
-- Downstream messages
-- **************************************************************************
CREATE TABLE lora_downstream_message (
    device_eui   CHAR(23) NOT NULL REFERENCES lora_device(eui) ON DELETE CASCADE,
    data         VARCHAR(256) NOT NULL,
    port         INTEGER NOT NULL,
    ack          BOOLEAN NOT NULL DEFAULT false,
    created_time INTEGER NOT NULL,
    sent_time    INTEGER DEFAULT 0,
    ack_time     INTEGER DEFAULT 0,

    CONSTRAINT lora_downstream_message_pk PRIMARY KEY (device_eui)
);

`

// Commands to purge the database
const purgeCommands string = `
DROP TABLE lora_downstream_message;
DROP TABLE lora_device_data;
DROP TABLE lora_device_nonce;
DROP TABLE lora_device;
DROP TABLE lora_application;
DROP TABLE lora_network;
DROP TABLE lora_sequence;
DROP TABLE lora_gateway;
DROP TABLE lora_token;
DROP TABLE lora_org_member;
DROP TABLE lora_owner;
DROP TABLE lora_organization;
DROP TABLE lora_user;
DROP TABLE lora_output;
`

func removeComments(schema string) string {
	ret := ""
	lines := strings.Split(schema, "\n")
	for _, v := range lines {
		line := v
		pos := strings.Index(line, "--")
		if pos == 0 {
			continue
		}
		if pos > 0 {
			line = line[0:pos]
		}
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		ret += line + "\n"
	}
	return ret
}

// SchemaCommandList returns a list of the DDL commands to create a schema.
func SchemaCommandList() []string {
	var ret []string

	commands := strings.Split(removeComments(DBSchema), ";")
	for _, v := range commands {
		if len(strings.TrimSpace(v)) > 0 {

			ret = append(ret, strings.TrimSpace(v))
		}
	}
	return ret
}
