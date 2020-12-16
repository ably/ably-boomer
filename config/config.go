package config

import "github.com/urfave/cli/v2"

type Config struct {
	TestType          string
	Env               string
	APIKey            string
	ChannelName       string
	PublishInterval   int
	NumSubscriptions  int
	MessageDataLength int
	Publisher         bool
	NumChannels       int
}

func Default() *Config {
	return &Config{
		TestType:          "",
		Env:               "",
		APIKey:            "",
		ChannelName:       "test_channel",
		PublishInterval:   10,
		NumSubscriptions:  2,
		MessageDataLength: 2000,
		Publisher:         false,
		NumChannels:       64,
	}
}

func (c *Config) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "test-type",
			Value:       c.TestType,
			Destination: &c.TestType,
			EnvVars:     []string{"ABLY_TEST_TYPE"},
			Usage:       "The type of load test to run. Can be either fanout, personal, sharded or composite.",
		},
		&cli.StringFlag{
			Name:        "env",
			Value:       c.Env,
			Destination: &c.Env,
			EnvVars:     []string{"ABLY_ENV"},
			Usage:       "The name of the Ably environment to run the load test against.",
		},
		&cli.StringFlag{
			Name:        "api-key",
			Value:       c.APIKey,
			Destination: &c.APIKey,
			EnvVars:     []string{"ABLY_API_KEY"},
			Usage:       "The API key to use.",
		},
		&cli.StringFlag{
			Name:        "channel-name",
			Value:       c.ChannelName,
			Destination: &c.ChannelName,
			EnvVars:     []string{"ABLY_CHANNEL_NAME"},
			Usage:       "The name of the channel to use. Only used for fanout type tests.",
		},
		&cli.IntFlag{
			Name:        "publish-interval",
			Value:       c.PublishInterval,
			Destination: &c.PublishInterval,
			EnvVars:     []string{"ABLY_PUBLISH_INTERVAL"},
			Usage:       "The number of milliseconds to wait between publishing messages.",
		},
		&cli.IntFlag{
			Name:        "num-subscriptions",
			Value:       c.NumSubscriptions,
			Destination: &c.NumSubscriptions,
			EnvVars:     []string{"ABLY_NUM_SUBSCRIPTIONS"},
			Usage:       "The number of subscriptions to create per channel.",
		},
		&cli.IntFlag{
			Name:        "msg-data-length",
			Value:       c.MessageDataLength,
			Destination: &c.MessageDataLength,
			EnvVars:     []string{"ABLY_MSG_DATA_LENGTH"},
			Usage:       "The number of characters to publish as message data.",
		},
		&cli.IntFlag{
			Name:        "num-channels",
			Value:       c.NumChannels,
			Destination: &c.NumChannels,
			EnvVars:     []string{"ABLY_NUM_CHANNELS"},
			Usage:       "The number of channels a worker could subscribe to. A channel will be chosen at random.",
		},
		&cli.BoolFlag{
			Name:        "publisher",
			Value:       c.Publisher,
			Destination: &c.Publisher,
			EnvVars:     []string{"ABLY_PUBLISHER"},
			Usage:       "If true, the worker will publish messages to the channels. If false, the worker will subscribe to the channels.",
		},
	}
}
