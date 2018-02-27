#!/bin/bash
set -e
mkdir -p congress/server/ui
mkdir -p postgresql/init
cd ../..
echo Building congress
go test > /dev/null 2>&1
go build 
#
# Generate the schema and initial data
#
TOKEN=$(hexdump -n 16 -e '"%08x"'  /dev/random)
./congress --printschema > deployment/docker/postgresql/init/congress.sql
cat > deployment/docker/postgresql/init/initdata.sql << EOF
insert into lora_token (
    user_id, 
    token, 
    resource, 
    write) 
values (
    'system', 
    '${TOKEN}', 
    '/', 
    true);
EOF
echo Building images...
GOOS=linux go build
cp congress deployment/docker/congress/server
cp ui/*.html deployment/docker/congress/server/ui
cp ui/*.css deployment/docker/congress/server/ui

cd deployment/docker
docker build -q congress --tag telenordigital:congress
docker build -q postgresql --tag telenordigital:congressdb
echo ${TOKEN} > congress.apitoken
echo "Ready to run docker-compose"

mkdir -p mqtt-broker/mosquitto/data
mkdir -p mqtt-broker/mosquitto/log

