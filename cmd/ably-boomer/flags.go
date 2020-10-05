package main

import "github.com/urfave/cli/v2"

var (
	// Ably.
	envFlag = &cli.StringFlag{
		Name:    "env",
		EnvVars: []string{"ABLY_ENV"},
		Usage:   "The name of the Ably environment to run the load test against.",
	}
	apiKeyFlag = &cli.StringFlag{
		Name:    "api-key",
		EnvVars: []string{"ABLY_API_KEY"},
		Usage:   "The API key to use.",
	}
	channelNameFlag = &cli.StringFlag{
		Name:    "channel-name",
		EnvVars: []string{"ABLY_CHANNEL_NAME"},
		Value:   "test_channel",
		Usage:   "The name of the channel to use.",
	}
	publishIntervalFlag = &cli.IntFlag{
		Name:    "publish-interval",
		EnvVars: []string{"ABLY_PUBLISH_INTERVAL"},
		Value:   10,
		Usage:   "The number of seconds to wait between publishing messages.",
	}
	numSubscriptionsFlag = &cli.IntFlag{
		Name:    "num-subscriptions",
		EnvVars: []string{"ABLY_NUM_SUBSCRIPTIONS"},
		Value:   2,
		Usage:   "The number of subscriptions to create per channel.",
	}
	msgDataLengthFlag = &cli.IntFlag{
		Name:    "msg-data-length",
		EnvVars: []string{"ABLY_MSG_DATA_LENGTH"},
		Value:   2000,
		Usage:   "The number of characters to publish as message data.",
	}
	numChannelsFlag = &cli.IntFlag{
		Name:    "num-channels",
		EnvVars: []string{"ABLY_NUM_CHANNELS"},
		Value:   64,
		Usage:   "The number of channels a worker could subscribe to. A channel will be chosen at random.",
	}
	sseSubscriberFlag = &cli.BoolFlag{
		Name:    "sse-subscriber",
		EnvVars: []string{"ABLY_SSE_SUBSCRIBER"},
		Value:   false,
		Usage:   "Whether to subscribe using SSE.",
	}

	// Perf.
	cpuProfileDirFlag = &cli.PathFlag{
		Name:    "cpu-profile-dir",
		EnvVars: []string{"PERF_CPU_PROFILE_DIR"},
		Usage:   "The directorty path to write the pprof cpu profile.",
	}
	s3BucketFlag = &cli.StringFlag{
		Name:    "s3-bucket",
		EnvVars: []string{"PERF_S3_BUCKET"},
		Usage:   "The name of the s3 bucket to upload pprof data to.",
	}

	// Boomer.
	boomerArgsFlag = &cli.StringSliceFlag{
		Name: "boomer",
		Value: cli.NewStringSlice(
			"-master-version-0.9.0",
			"-master-host", "127.0.0.1",
			"-master-port", "5557"),
	}
)
