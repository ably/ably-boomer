package main

import "github.com/urfave/cli/v2"

var (
	// Ably
	testTypeFlag = &cli.StringFlag{
		Name:  "test-type",
		Usage: "The type of load test to run. Can be either fanout, personal, sharded or composite.",
	}
	envFlag = &cli.StringFlag{
		Name:  "env",
		Usage: "The name of the Ably environment to run the load test against.",
	}
	apiKeyFlag = &cli.StringFlag{
		Name:  "api-key",
		Usage: "The API key to use.",
	}
	channelNameFlag = &cli.StringFlag{
		Name:  "channel-name",
		Value: "test_channel",
		Usage: "The name of the channel to use. Only used for fanout type tests.",
	}
	publishIntervalFlag = &cli.IntFlag{
		Name:  "publish-interval",
		Value: 10,
		Usage: "The number of seconds to wait between publishing messages. Only used for personal, sharded and composite type tests.",
	}
	numSubscriptionsFlag = &cli.IntFlag{
		Name:  "num-subscriptions",
		Value: 2,
		Usage: "The number of subscriptions to create per channel. Only used for personal, sharded and composite type tests.",
	}
	msgDataLengthFlag = &cli.IntFlag{
		Name:  "msg-data-length",
		Value: 2000,
		Usage: "The number of characters to publish as message data. Only used for personal, sharded and composite type tests.",
	}
	publisherFlag = &cli.BoolFlag{
		Name:  "publisher",
		Usage: "If true, the worker will publish messages to the channels. If false, the worker will subscribe to the channels. Only used for sharded type tests.",
	}
	numChannelsFlag = &cli.IntFlag{
		Name:  "num-channels",
		Value: 64,
		Usage: "The number of channels a worker could subscribe to. A channel will be chosen at random. Only used for sharded and composite type tests.",
	}

	// Perf
	cpuProfileDirFlag = &cli.PathFlag{
		Name:  "cpu-profile-dir",
		Usage: "The directorty path to write the pprof cpu profile.",
	}
	s3BucketFlag = &cli.StringFlag{
		Name:  "s3-bucket",
		Usage: "The name of the s3 bucket to upload pprof data to.",
	}

	// AWS
	regionFlag = &cli.StringFlag{
		Name:  "region",
		Usage: "The AWS region to use, i.e. us-west-2.",
	}
	sdkLoadConfigFlag = &cli.BoolFlag{
		Name:  "sdk-load-config",
		Usage: "A boolean indicating that region should be read from config in ~/.aws.",
	}
	profileFlag = &cli.StringFlag{
		Name:  "profile",
		Usage: "The AWS profile to use in the shared credentials file.",
	}
	accessKeyIDFlag = &cli.StringFlag{
		Name:  "access-key-id",
		Usage: "The AWS access key id credential to use.",
	}
	secretAccessKeyFlag = &cli.StringFlag{
		Name:  "secret-access-key ",
		Usage: " The AWS secret access key to use.",
	}
	sessionTokenFlag = &cli.StringFlag{
		Name:  "session-token",
		Usage: "The AWS session token to use.",
	}
)
