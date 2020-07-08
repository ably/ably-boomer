# ably-boomer

Ably load generator for Locust.

Ably-boomer will listen and report to a Locust master.


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
             -e "ABLY_TEST_TYPE=<fanout | personal>" \
             --ulimit nofile=250000:250000 \
             --rm ably-boomer \
             --master-version-0.9.0 \
             --master-host=<host address>
```

### Test Types

Different test types will simulate different usage patterns.

#### Fanout

A Fanout type test will simulate a single channel with a large number of subscribers.

Each Locust user will create a single subscription.

No messages will be published to the channel - this will need to be done separately.

#### Personal

A Personal type test will simulate a large number of channels, each with a small amount of subscribers.

Each Locust user will create a new channel with some subscribers.

Messages will be published to the channel at a regular interval.

### Test Configuration

The test is configured through environment variables.

Variable | Description | Default | Required
--- | --- | --- | ---
`ABLY_TEST_TYPE` | The type of load test to run. Can be either `fanout` or `personal`. | n/a | yes
`ABLY_ENV` | The name of the Ably environment to run the load test against. | n/a | yes
`ABLY_API_KEY` | The API key to use. | n/a | yes
`ABLY_CHANNEL_NAME` | The name of the channel to use. Only used for `fanout` type tests. | `test_channel` | no
`ABLY_PUBLISH_INTERVAL` | The number of seconds to wait between publishing messages. Only used for `personal` type tests. | `10` | no
`ABLY_NUM_SUBSCRIPTIONS` | The number of subscriptions to create per channel. Only used for `personal` type tests. | `2` | no
`ABLY_MSG_DATA_LENGTH` | The number of characters to publish as message data. Only used for `personal` type tests. | `2000` | no
