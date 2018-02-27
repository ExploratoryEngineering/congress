#!/bin/bash

#
# If the token becomes out of sync with the one stored in the database run this script to retrieve the correct one.
# The token might be out of sync if the container is rebuilt without changes.
echo "select token from lora_token;"| PGPASSWORD=thepassword psql -U congress -d congress -h localhost -t > congress.apitoken
