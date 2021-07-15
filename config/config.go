package config

import (
	"time"

	"github.com/ably/ably-boomer/perf"
	"github.com/ably/ably-go/ably"
	"github.com/docker/go-units"
	"github.com/inconshreveable/log15"
)

func New() *Config {
	return &Config{}
}

func Default() *Config {
	conf := New()

	conf.Client = ClientAbly

	conf.Subscriber.Enabled = false
	conf.Subscriber.Channels = "ably-boomer-test"
	conf.Subscriber.PushDevice = SubscriberPushDeviceConfig{
		Enabled:            false,
		URL:                "https://rest.ably.io",
		MetachannelEnabled: false,
	}

	conf.Publisher.Enabled = false
	conf.Publisher.Channels = "ably-boomer-test"
	conf.Publisher.PublishInterval = time.Second
	conf.Publisher.MessageSize = 2 * units.KiB
	conf.Publisher.PushEnabled = false

	conf.Standalone.Enabled = false
	conf.Standalone.Users = 1
	conf.Standalone.SpawnRate = 1

	conf.Locust.Host = "127.0.0.1"
	conf.Locust.Port = 5557

	conf.Ably.ConnectionTimeout = 4 * time.Second
	conf.Ably.RequestTimeout = 10 * time.Second
	conf.Ably.ChannelModes = "" // Use default modes.

	conf.Log.Level = log15.LvlInfo.String()

	conf.Redis.Enabled = false
	conf.Redis.Addr = "127.0.0.1:6379"
	conf.Redis.ConnectTimeout = 5 * time.Second
	conf.Redis.WorkerNumberKey = "ably-boomer:worker-number"

	return conf
}

type Config struct {
	Client     string
	Subscriber SubscriberConfig
	Publisher  PublisherConfig
	Presence   PresenceConfig
	Standalone StandaloneConfig
	Locust     LocustConfig
	Ably       AblyConfig
	Perf       perf.Conf
	Log        LogConfig
	Redis      RedisConf
	Custom     interface{}
}

const (
	ClientAbly    = "ably"
	ClientAblySSE = "ably-sse"
	ClientCustom  = "custom"
)

type SubscriberConfig struct {
	Enabled           bool
	Channels          string
	ReconnectInterval time.Duration
	PushDevice        SubscriberPushDeviceConfig
}

type SubscriberPushDeviceConfig struct {
	Enabled                    bool
	URL                        string
	MetachannelEnabled         bool
	RegistrationUpdateInterval time.Duration
	SubscriptionUpdateInterval time.Duration
}

type PublisherConfig struct {
	Enabled         bool
	Channels        string
	PublishInterval time.Duration
	MessageSize     int64
	PushEnabled     bool
}

type PresenceConfig struct {
	Enabled  bool
	Channels string
}

type StandaloneConfig struct {
	Enabled   bool
	Users     int
	SpawnRate float64
}

type LocustConfig struct {
	Host string
	Port int
}

type AblyConfig struct {
	APIKey            string
	Environment       string
	ConnectionTimeout time.Duration
	RequestTimeout    time.Duration
	ChannelModes      string
}

func (a *AblyConfig) ClientOptions() []ably.ClientOption {
	opts := []ably.ClientOption{
		ably.WithKey(a.APIKey),
		// Set the connection and request timeouts for REST.
		ably.WithHTTPRequestTimeout(a.RequestTimeout),
		// Set the connection timeout for Realtime.
		ably.WithHTTPOpenTimeout(a.ConnectionTimeout),
		// Set the request timeout for Realtime.
		ably.WithRealtimeRequestTimeout(a.RequestTimeout),
	}
	if a.Environment != "" {
		opts = append(opts, ably.WithEnvironment(a.Environment))
	}
	return opts
}

type LogConfig struct {
	Level string
}

type RedisConf struct {
	Enabled         bool
	Addr            string
	ConnectTimeout  time.Duration
	WorkerNumberKey string
}
