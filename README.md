# ably-boomer

Ably load generator for Locust.

Ably-boomer will listen and report to the Locust master.

Any host address specified in the Locust web UI will be ignored.

## Usage

To build the Docker container:

```bash
$ make image
```

To run the Docker container:

```bash
$ docker run -e "ABLY_ENV=<env>" \
             -e "ABLY_API_KEY=<api key>" \
             -e "ABLY_CHANNEL_NAME=<channel name>" \
             --rm <container id> \
             --master-host=<host address>
```
