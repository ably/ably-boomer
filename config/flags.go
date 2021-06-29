package config

import (
	"fmt"
	"os"

	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

var DefaultConfigPath = "ably-boomer.yaml"

func (c *Config) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to the config file",
			Value:   DefaultConfigPath,
			EnvVars: []string{"CONFIG_PATH"},
		},
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "client",
			Usage:       "The type of client to use",
			Value:       c.Client,
			Destination: &c.Client,
			EnvVars:     []string{"CLIENT"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "subscriber.enabled",
			Usage:       "Run subscribers",
			Value:       c.Subscriber.Enabled,
			Destination: &c.Subscriber.Enabled,
			EnvVars:     []string{"SUBSCRIBER_ENABLED"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "subscriber.channels",
			Usage:       "The channels to subscribe each user to (comma separated)",
			Value:       c.Subscriber.Channels,
			Destination: &c.Subscriber.Channels,
			EnvVars:     []string{"SUBSCRIBER_CHANNELS"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "subscriber.push-device.enabled",
			Usage:       "Register and subscribe a push device",
			Value:       c.Subscriber.PushDevice.Enabled,
			Destination: &c.Subscriber.PushDevice.Enabled,
			EnvVars:     []string{"SUBSCRIBER_PUSH_DEVICE_ENABLED"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "subscriber.push-device.url",
			Usage:       "The REST URL that should be used by the AblyChannel push devices to publish",
			Value:       c.Subscriber.PushDevice.URL,
			Destination: &c.Subscriber.PushDevice.URL,
			EnvVars:     []string{"SUBSCRIBER_PUSH_DEVICE_URL"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "subscriber.push-device.metachannel-enabled",
			Usage:       "Subscribe to the push metachannel to receive push delivery errors",
			Value:       c.Subscriber.PushDevice.MetachannelEnabled,
			Destination: &c.Subscriber.PushDevice.MetachannelEnabled,
			EnvVars:     []string{"SUBSCRIBER_PUSH_DEVICE_METACHANNEL_ENABLED"},
		}),
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:        "subscriber.push-device.registration-update-interval",
			Usage:       "The interval between two registration updates (0 to disable)",
			Value:       c.Subscriber.PushDevice.RegistrationUpdateInterval,
			Destination: &c.Subscriber.PushDevice.RegistrationUpdateInterval,
			EnvVars:     []string{"SUBSCRIBER_PUSH_DEVICE_REGISTRATION_UPDATE_INTERVAL"},
		}),
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:        "subscriber.push-device.subscription-update-interval",
			Usage:       "The interval between two subscription updates (0 to disable)",
			Value:       c.Subscriber.PushDevice.SubscriptionUpdateInterval,
			Destination: &c.Subscriber.PushDevice.SubscriptionUpdateInterval,
			EnvVars:     []string{"SUBSCRIBER_PUSH_DEVICE_SUBSCRIPTION_UPDATE_INTERVAL"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "publisher.enabled",
			Usage:       "Run publishers",
			Value:       c.Publisher.Enabled,
			Destination: &c.Publisher.Enabled,
			EnvVars:     []string{"PUBLISHER_ENABLED"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "publisher.channels",
			Usage:       "The channels each user should publish messages to (comma separated)",
			Value:       c.Publisher.Channels,
			Destination: &c.Publisher.Channels,
			EnvVars:     []string{"PUBLISHER_CHANNELS"},
		}),
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:        "publisher.publish-interval",
			Usage:       "The interval between publishes to each channel by each user",
			Value:       c.Publisher.PublishInterval,
			Destination: &c.Publisher.PublishInterval,
			EnvVars:     []string{"PUBLISHER_PUBLISH_INTERVAL"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "publisher.push-enabled",
			Usage:       "Publish onto a push-enabled channel",
			Value:       c.Publisher.PushEnabled,
			Destination: &c.Publisher.PushEnabled,
			EnvVars:     []string{"PUBLISHER_PUSH_ENABLED"},
		}),
		altsrc.NewInt64Flag(&cli.Int64Flag{
			Name:        "publisher.message-size",
			Usage:       "The size of messages published by each user",
			Value:       c.Publisher.MessageSize,
			Destination: &c.Publisher.MessageSize,
			EnvVars:     []string{"PUBLISHER_MESSAGE_SIZE"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "presence.enabled",
			Usage:       "Run presence users",
			Value:       c.Presence.Enabled,
			Destination: &c.Presence.Enabled,
			EnvVars:     []string{"PRESENCE_ENABLED"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "presence.channels",
			Usage:       "The channels each user should be present in (comma separated)",
			Value:       c.Presence.Channels,
			Destination: &c.Presence.Channels,
			EnvVars:     []string{"PRESENCE_CHANNELS"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "standalone.enabled",
			Aliases:     []string{"standalone", "s"},
			Usage:       "Run ably-boomer in standalone mode (i.e. without Locust)",
			Value:       c.Standalone.Enabled,
			Destination: &c.Standalone.Enabled,
			EnvVars:     []string{"STANDALONE_ENABLED"},
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "standalone.users",
			Usage:       "Number of users to run when running in standalone mode",
			Value:       c.Standalone.Users,
			Destination: &c.Standalone.Users,
			EnvVars:     []string{"STANDALONE_USERS"},
		}),
		altsrc.NewFloat64Flag(&cli.Float64Flag{
			Name:        "standalone.spawn-rate",
			Usage:       "Number of users to spawn per second when running in standalone mode",
			Value:       c.Standalone.SpawnRate,
			Destination: &c.Standalone.SpawnRate,
			EnvVars:     []string{"STANDALONE_SPAWN_RATE"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "locust.host",
			Usage:       "Locust master host",
			Value:       c.Locust.Host,
			Destination: &c.Locust.Host,
			EnvVars:     []string{"LOCUST_HOST"},
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "locust.port",
			Usage:       "Locust master port",
			Value:       c.Locust.Port,
			Destination: &c.Locust.Port,
			EnvVars:     []string{"LOCUST_PORT"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "ably.api-key",
			Usage:       "The Ably API key to use",
			Value:       c.Ably.APIKey,
			Destination: &c.Ably.APIKey,
			EnvVars:     []string{"ABLY_API_KEY"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "ably.env",
			Usage:       "The name of the Ably environment to run the load test against",
			Value:       c.Ably.Environment,
			Destination: &c.Ably.Environment,
			EnvVars:     []string{"ABLY_ENV"},
		}),
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:        "ably.connection-timeout",
			Usage:       "The connection timeout",
			Value:       c.Ably.ConnectionTimeout,
			Destination: &c.Ably.ConnectionTimeout,
			EnvVars:     []string{"ABLY_CONNECTION_TIMEOUT"},
		}),
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:        "ably.request-timeout",
			Usage:       "The request timeout",
			Value:       c.Ably.RequestTimeout,
			Destination: &c.Ably.RequestTimeout,
			EnvVars:     []string{"ABLY_REQUEST_TIMEOUT"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "ably.channel-modes",
			Usage:       "The Channel Modes to use (comma separated, set to empty to use the default modes) Valid modes are 'presence', 'publish', 'subscribe', and 'presenceSubscribe'",
			Value:       c.Ably.ChannelModes,
			Destination: &c.Ably.ChannelModes,
			EnvVars:     []string{"ABLY_CHANNEL_MODES"},
		}),
		altsrc.NewPathFlag(&cli.PathFlag{
			Name:        "perf.cpu-profile-dir",
			Usage:       "The directory path to write the pprof cpu profile",
			Value:       c.Perf.CPUProfileDir,
			Destination: &c.Perf.CPUProfileDir,
			EnvVars:     []string{"PERF_CPU_PROFILE_DIR"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "perf.s3-bucket",
			Usage:       "The name of the s3 bucket to upload pprof data to",
			Value:       c.Perf.S3Bucket,
			Destination: &c.Perf.S3Bucket,
			EnvVars:     []string{"PERF_S3_BUCKET"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "log.level",
			Usage:       "The log level",
			Value:       c.Log.Level,
			Destination: &c.Log.Level,
			EnvVars:     []string{"LOG_LEVEL"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "redis.enabled",
			Usage:       "Use Redis to assign incremental worker numbers",
			Value:       c.Redis.Enabled,
			Destination: &c.Redis.Enabled,
			EnvVars:     []string{"REDIS_ENABLED"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "redis.addr",
			Usage:       "Redis TCP address",
			Value:       c.Redis.Addr,
			Destination: &c.Redis.Addr,
			EnvVars:     []string{"REDIS_ADDR"},
		}),
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:        "redis.connect-timeout",
			Usage:       "Redis connection timeout",
			Value:       c.Redis.ConnectTimeout,
			Destination: &c.Redis.ConnectTimeout,
			EnvVars:     []string{"REDIS_CONNECT_TIMEOUT"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "redis.worker-number-key",
			Usage:       "Redis key to use to assign worker numbers",
			Value:       c.Redis.WorkerNumberKey,
			Destination: &c.Redis.WorkerNumberKey,
			EnvVars:     []string{"REDIS_WORKER_NUMBER_KEY"},
		}),
	}
}

func InitFileSourceFunc(flags []cli.Flag, log log15.Logger) func(*cli.Context) error {
	return func(c *cli.Context) error {
		path := c.String("config")
		if path == "" {
			return nil
		}
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			log.Warn(fmt.Sprintf("config file not found: %s, using CLI args and env vars only", path))
			return nil
		} else if err != nil {
			return err
		}
		return altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("config"))(c)
	}
}
