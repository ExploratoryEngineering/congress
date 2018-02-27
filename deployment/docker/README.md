# Docker stack for congress

You'll obviously need a working Docker installation for this. No surprises there.

Build the required images and configuration by launching the `build-images.sh` script.
This will create two new images; `telenordigital:congress` and `telenordigital:congressdb`.

Launch the stack with `docker-compose start`, inspect the logs via the usual Docker 
command or through `docker-compose logs`.

The database image performs a number of initializations so the congress image might not start 
up the first time the stack is launched. Run `docker-compose up db` to initialize the database 
the first time or re-launch the stack to fix it.

The Congress API endpoint is at `http://localhost:8080/`. Query the database for the API token.
If this is the first time you've created the image the token is in the file `congress.apitoken`.

The token in the Docker image might not be the same as in the `congress.apitoken` file.
