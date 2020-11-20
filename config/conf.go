package config

import (
	"github.com/urfave/cli/v2"
)

var (
	// Ably.
	EnvFlag = &cli.StringFlag{
		Name:    "env",
		EnvVars: []string{"ABLY_ENV"},
		Usage:   "The name of the Ably environment to run the load test against.",
	}
	APIKeyFlag = &cli.StringFlag{
		Name:    "api-key",
		EnvVars: []string{"ABLY_API_KEY"},
		Usage:   "The API key to use.",
	}
	PublishIntervalFlag = &cli.IntFlag{
		Name:    "publish-interval",
		EnvVars: []string{"ABLY_PUBLISH_INTERVAL"},
		Value:   10,
		Usage:   "The number of milliseconds to wait between publishing messages.",
	}
	NumSubscriptionsFlag = &cli.IntFlag{
		Name:    "num-subscriptions",
		EnvVars: []string{"ABLY_NUM_SUBSCRIPTIONS"},
		Value:   2,
		Usage:   "The number of subscriptions to create per channel.",
	}
	MsgDataLengthFlag = &cli.IntFlag{
		Name:    "msg-data-length",
		EnvVars: []string{"ABLY_MSG_DATA_LENGTH"},
		Value:   2000,
		Usage:   "The number of characters to publish as message data.",
	}
	NumChannelsFlag = &cli.IntFlag{
		Name:    "num-channels",
		EnvVars: []string{"ABLY_NUM_CHANNELS"},
		Value:   64,
		Usage:   "The number of channels a worker could subscribe to. A channel will be chosen at random.",
	}
	SSESubscriberFlag = &cli.BoolFlag{
		Name:    "sse-subscriber",
		EnvVars: []string{"ABLY_SSE_SUBSCRIBER"},
		Value:   false,
		Usage:   "Whether to subscribe using SSE.",
	}

	// Perf.
	CPUProfileDirFlag = &cli.PathFlag{
		Name:    "cpu-profile-dir",
		EnvVars: []string{"PERF_CPU_PROFILE_DIR"},
		Usage:   "The directory path to write the pprof cpu profile.",
	}
	S3BucketFlag = &cli.StringFlag{
		Name:    "s3-bucket",
		EnvVars: []string{"PERF_S3_BUCKET"},
		Usage:   "The name of the s3 bucket to upload pprof data to.",
	}

	LocustHostFlag = &cli.StringFlag{
		Name:    "locust-host",
		EnvVars: []string{"LOCUST_HOST"},
		Usage:   "The hostname of the locust instance.",
		Value:   "127.0.0.1",
	}
	LocustPortFlag = &cli.IntFlag{
		Name:    "locust-port",
		Usage:   "The port of the locust instance.",
		EnvVars: []string{"LOCUST_PORT"},
		Value:   5557,
	}
)
var TaskFlags  = []cli.Flag{
	EnvFlag,
	APIKeyFlag,
	PublishIntervalFlag,
	NumSubscriptionsFlag,
	MsgDataLengthFlag,
	NumChannelsFlag,
	SSESubscriberFlag,
	CPUProfileDirFlag,
	S3BucketFlag,
}

var CommonFlags = []cli.Flag{
	LocustHostFlag,
	LocustPortFlag,
}

// Conf is the task's configuration.
type Conf struct {
	APIKey           string
	Env              string
	NumChannels      int
	MsgDataLength    int
	SSESubscriber    bool
	NumSubscriptions int
	PublishInterval  int
}

func ParseConf(c *cli.Context) Conf {
	return Conf{
		APIKey:           c.String(APIKeyFlag.Name),
		Env:              c.String(EnvFlag.Name),
		NumChannels:      c.Int(NumChannelsFlag.Name),
		MsgDataLength:    c.Int(MsgDataLengthFlag.Name),
		SSESubscriber:    c.Bool(SSESubscriberFlag.Name),
		NumSubscriptions: c.Int(NumSubscriptionsFlag.Name),
		PublishInterval:  c.Int(PublishIntervalFlag.Name),
	}
}
