# PostgreSQL database

The base image is the default PostgreSQL image. Connect to the server via `psql`like so:

    psql -U congress -d congress -h localhost

The API token is created only once, when the image is built. Run the following query
to read the token:

   select * from lora_token

