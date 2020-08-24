# ably-boomer

Ably load generator for Locust, based on the [boomer](https://github.com/myzhan/boomer) library.

Ably-boomer creates Ably realtime connections and subscribes to channels, in order to generate load, and measure delivery and latency statistics, reporting back to a Locust master.

## Usage

Ably-boomer by default runs against Locust 1.1, but supports compatibility with Locust 0.9.0 via a command-line option.

To run the Docker container against a Locust 0.9.0 master:

```bash
$ docker run -e "ABLY_ENV=<env>" \
             -e "ABLY_API_KEY=<api key>" \
             -e "ABLY_TEST_TYPE=<fanout | personal | sharded | composite>" \
             --ulimit nofile=250000:250000 \
             --rm ablyrealtime/ably-boomer \
             --master-version-0.9.0 \
             --master-host=<host address>
```

## Test Types

Different test types will simulate different usage patterns.

### Fanout

A Fanout type test will simulate a single channel with a large number of subscribers.

Each Locust user will create a single subscription.

No messages will be published to the channel - this will need to be performed separately.

### Personal

A Personal type test will simulate a large number of channels, each with a small number of subscribers.

Each Locust user will create a new channel with a randomly-generated name with a configurable number of subscriber connections for that channel.

The Ably-boomer user publishes messages to the channel periodically with a configurable interval.

### Sharded

A Sharded type test will simulate a large number of subscribers, sharded over a number of channels.

Each Locust user will create a number of subscribers, distributed evenly over the configured number of channels.

You can publish messages to these channels by running a task with the `ABLY_PUBLISHER` environment variable set to `true`.

### Composite

A Composite type test will simulate both a Personal scenario and a Sharded scenario.

Each Locust user will create a single connection with a single subcription to a Sharded Fanout channel, as well as a number of subscriptions to a Personal channel.

You can publish messages to the Sharded Fanout channels by running a Sharded Fanout task with the `ABLY_PUBLISHER` environment variable set to `true`.

## Test Configuration

The test is configured through environment variables.

Variable | Description | Default | Required
--- | --- | --- | ---
`ABLY_TEST_TYPE` | The type of load test to run. Can be either `fanout`, `personal`, `sharded` or `composite`. | n/a | yes
`ABLY_ENV` | The name of the Ably environment to run the load test against. | n/a | yes
`ABLY_API_KEY` | The API key to use. | n/a | yes
`ABLY_CHANNEL_NAME` | The name of the channel to use. Only used for `fanout` type tests. | `test_channel` | no
`ABLY_PUBLISH_INTERVAL` | The number of seconds to wait between publishing messages. Only used for `personal`, `sharded` and `composite` type tests. | `10` | no
`ABLY_NUM_SUBSCRIPTIONS` | The number of subscriptions to create per channel. Only used for `personal`, `sharded` and `composite` type tests. | `2` | no
`ABLY_MSG_DATA_LENGTH` | The number of characters to publish as message data. Only used for `personal`, `sharded` and `composite` type tests. | `2000` | no
`ABLY_PUBLISHER` | If `true`, the worker will publish messages to the channels. If `false`, the worker will subscribe to the channels. Only used for `sharded` type tests. | `false` | no
`ABLY_NUM_CHANNELS` | The number of channels a worker could subscribe to. A channel will be chosen at random. Only used for `sharded` and `composite` type tests. | `64` | no

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




## Build

Ably-boomer is a go executable which may be packaged as a Docker container.

To build the Docker container:

```bash
$ make image
```

To compile the Go executable:

```bash
$ make build
```
