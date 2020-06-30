# ably-boomer

Ably load generator for Locust.

Ably-boomer will listen and report to the Locust master.

Any host address specified in the Locust web UI will be ignored.

## Build

To build the Docker container:

```bash
$ make image
```

## Usage

To run the Docker container against a Locust 0.9.0 master:

```bash
$ docker run -e "ABLY_ENV=<env>" \
             -e "ABLY_API_KEY=<api key>" \
             -e "ABLY_CHANNEL_NAME=<channel name>" \
             --rm ably-boomer \
             --master-version-0.9.0 \
             --master-host=<host address>
```

### Test Configuration

The test is configured through environment variables.

Variable | Description | Default | Required
--- | --- | --- | ---
`ABLY_ENV` | The name of the Ably environment to run the load test against. | n/a | yes
`ABLY_API_KEY` | The API key to use. | n/a | yes
`ABLY_CHANNEL_NAME` | The name of the channel to use. | n/a | yes
