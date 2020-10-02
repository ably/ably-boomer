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
		Usage:   "The name of the channel to use. Only used for fanout type tests.",
	}
	publishIntervalFlag = &cli.IntFlag{
		Name:    "publish-interval",
		EnvVars: []string{"ABLY_PUBLISH_INTERVAL"},
		Value:   10,
		Usage:   "The number of seconds to wait between publishing messages. Only used for personal, sharded and composite type tests.",
	}
	numSubscriptionsFlag = &cli.IntFlag{
		Name:    "num-subscriptions",
		EnvVars: []string{"ABLY_NUM_SUBSCRIPTIONS"},
		Value:   2,
		Usage:   "The number of subscriptions to create per channel. Only used for personal, sharded and composite type tests.",
	}
	msgDataLengthFlag = &cli.IntFlag{
		Name:    "msg-data-length",
		EnvVars: []string{"ABLY_MSG_DATA_LENGTH"},
		Value:   2000,
		Usage:   "The number of characters to publish as message data. Only used for personal, sharded and composite type tests.",
	}
	publisherFlag = &cli.BoolFlag{
		Name:    "publisher",
		EnvVars: []string{"ABLY_PUBLISHER"},
		Usage:   "If true, the worker will publish messages to the channels. If false, the worker will subscribe to the channels. Only used for sharded type tests.",
	}
	numChannelsFlag = &cli.IntFlag{
		Name:    "num-channels",
		EnvVars: []string{"ABLY_NUM_CHANNELS"},
		Value:   64,
		Usage:   "The number of channels a worker could subscribe to. A channel will be chosen at random. Only used for sharded and composite type tests.",
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
)
