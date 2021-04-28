# ablyboomer

ablyboomer is an Ably load generator for Locust written in Go, based on the [boomer](https://github.com/myzhan/boomer) library.

ablyboomer defines a worker that performs the same function as a [Locust worker](https://docs.locust.io/en/stable/running-locust-distributed.html) in a distributed load test. It receives start and stop events from the Locust master, spawns the appropriate amount of users that subscribe, publish and enter Ably channels based on how ablyboomer is configured, and reports message delivery and latency statistics back to the master.

## Quick Start

Follow these steps to run a simple fanout load test against the Ably service:

- Build the Docker image:

```
make image
```

- Copy `.env.example` to `.env` to store your private environment variables:

```
cp .env.example .env
```

- Set `ABLY_API_KEY` in `.env` to your Ably app key

- Start the docker-compose cluster in `examples/fanout`:

```
cd examples/fanout
docker-compose up
```

- Visit http://localhost:8089 in your browser and start a load test with 50 users spawning at 5 users/sec

You are now running 5 ablyboomer workers that each have 10 users subscribed to a `fanout` channel, and also
have a single standalone worker publishing a message to the `fanout` channel every 1s, with message latency
visible in the Locust web interface.

## Building

To build ablyboomer as a local binary, run:

```
make build
```

This builds a binary at `bin/ably-boomer` that you can run directly:

```
bin/ably-boomer --help
```

You can also build an ablyboomer Docker image with:

```
make image
```

Which is tagged as `ablyrealtime/ably-boomer:latest` and can be run like the binary:

```
docker run --rm ablyrealtime/ably-boomer:latest --help
```

## Config

ablyboomer can be configured either using CLI flags, environment variables or a YAML config file, with
precedence going from config file -> environment variables -> CLI flags.

The path to the YAML config file defaults to `ably-boomer.yaml` in the current directory but can be set
explicitly with the `--config` CLI flag:

```
bin/ably-boomer --config my-config.yaml
```

An example YAML config file for a fanout subscriber load test:

```yaml
locust.host: locust.example.com
subscriber.enabled: true
subscriber.channels: fanout
```

Or for a standalone publisher that doesn't connect to Locust but just publishes messages to the fanout channel:

```yaml
standalone.enabled: true
standalone.users: 1
standalone.spawn-rate: 1.0
publisher.enabled: true
publisher.channels: fanout
publisher.publish-interval: 1s
```

See `bin/ably-boomer --help` for a full list of config options.

### User Numbering

When running more than one ablyboomer process, Redis can be used to assign a unique number to each user
in the load test. For example, with Redis enabled, if you have 5 workers that each start 10 users, they
will use Redis to assign numbers 1-10, 11-20, 21-30, 31-40 and 41-50.

Redis can be configured with:

```yaml
redis.enable: true
redis.addr: redis-host:6379
redis.connect-timeout: 5s
```

### Channel Config

The `subscriber.channels`, `publisher.channels` and `presence.channels` config options are a comma separated
list of channels that a user should subscribe to, publish to or enter respectively.

These config options can include a Go template to dynamically render the names of the channels, with all the
[built-in functions](https://golang.org/pkg/text/template/#hdr-Functions) available as well as a `.UserNumber`
variable which is the 1-indexed number of the current user and a `mod` function which performs a modulo calculation
between two integers.

For example, to generate a "personal" channel name that is unique to each user:

```yaml
subscriber.channels: personal-{{ .UserNumber }}
```

Or to do the same but with left-padded zeros:

```yaml
subscriber.channels: personal-{{ printf "%08d" .UserNumber }}
```

Or to generate a "sharded" channel name that spreads the users over a fixed set of 10 channels:

```yaml
subscriber.channels: sharded-{{ mod .UserNumber 10 }}
```

## Examples

See the `examples` directory for some example load tests which can be run using docker-compose.

Each example can be run by changing into the directory, running `docker-compose up` and visiting
http://localhost:8089 in your browser:

```
cd examples/personal

docker-compose up

# now visit http://localhost:8089
```

### Fanout

The fanout example simulates a single channel with a large number of subscribers.

Each user creates a single subscription.

A standalone publisher publishes 1 message per second.

### Personal

The personal example simulates a large number of channels, each with one subscriber.

Each user subscribes to a channel based on their assigned number (e.g. `personal-0042`), and publishes a message to it every second.

### Sharded

The sharded example simulates a large number of subscribers, sharded over a number of channels.

Each user subscribes to a channel using their assigned number modulo 5 (i.e. one of `sharded-0`, `sharded-1`, `sharded-2`, `sharded-3` or `sharded-5`).

A standalone publisher publishes 1 message per second to each of the 5 sharded channels.

### Composite

The composite example simulates both a personal scenario and a sharded scenario.

Each user subscribes to a personal channel (e.g. `personal-0042`), publishes a message to it every second,
and also subscribes to a sharded channel (i.e. one of `sharded-0`, `sharded-1`, `sharded-2`, `sharded-3` or `sharded-5`).

A standalone publisher publishes 1 message per second to each of the 5 sharded
channels.

### Push Fanout

The push fanout example simulates a single channel with a large number of push device subscribers.

Each user registers a push device  (e.g. `device-0042`) with the ablyChannel transport which publishes messages back to a channel (e.g. `push-0042`), and subscribes to that channel to measure latency of messages pushed to it. Each user then subscribes that push device to a single fanout channel.

A standalone publisher publishes 1 message per second to the fanout channel.

Note that running this test requires you to [enable push
notifications](https://knowledge.ably.com/what-are-channel-rules-and-how-can-i-use-them-in-my-app)
on a namespace (or to enable it in the default channel rule), and add it
to the subscriber and publisher channels config options. For example, with a namespace
called `push`:

```yaml
subscriber.channels: push:fanout

publisher.channels: push:fanout
```

## Performance Options

The test can be configured to debug performance. Options are set through environment variables.

Variable | Description | Default | Required
--- | --- | --- | ---
`PERF_CPU_PROFILE_DIR` | The directorty path to write the pprof cpu profile | n/a | no
`PERF_CPU_S3_BUCKET` | The name of the s3 bucket to upload pprof data to | n/a | no

If uploading data to s3, the s3 client is configured through the default environment as per the
[s3 client documentation](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html).


The `AWS_REGION` must be set by either specifying the `AWS_REGION` environment variable or to load the environment
from the configuration by setting `AWS_SDK_LOAD_CONFIG=true`. Credentials will be retrieved from `~/.aws` and a
profile can be selected by setting the `AWS_PROFILE` environment variable. If not using the credentials file, the
settings can be provided directly through environment variables.


Variable | Description | Default | Required
--- | --- | --- | ---
`AWS_REGION` | The AWS region to use, i.e. `us-west-2` | n/a | no
`AWS_SDK_LOAD_CONFIG` | A boolean indicating that region should be read from config in `~/.aws` | n/a | no
`AWS_PROFILE` | The aws profile to use in the shared credentials file | "default" | no
`AWS_ACCESS_KEY_ID` | The AWS access key id credential to use | n/a | no
`AWS_SECRET_ACCESS_KEY`| The AWS secret access key to use | n/a | no
`AWS_SESSION_TOKEN` | The AWS session token to use | n/a | no
