package main

import (
	"os"
	"strconv"
	"strings"
)

// DefaultChannelName defines the channel name used by default in test runs
const DefaultChannelName = "test_channel"

// DefaultPublishInterval defines the default time between publishing message
const DefaultPublishInterval = "10"

// DefaultNumSubscriptions defines the default number of subscibers in test runs
const DefaultNumSubscriptions = "2"

// DefaultMessageDataLength defines the default size of messages in test runs
const DefaultMessageDataLength = "2000"

// DefaultPublisher defines if the test run should publish by default
const DefaultPublisher = "false"

// DefaultNumChannels defines the default number of channels used in test runs
const DefaultNumChannels = "64"

// TestConfig defines the configuration for a specific test run
type TestConfig struct {
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

func newTestConfig() TestConfig {
	return TestConfig{
		TestType:          ablyTestType(),
		Env:               ablyEnv(),
		APIKey:            ablyAPIKey(),
		ChannelName:       ablyChannelName(),
		PublishInterval:   ablyPublishInterval(),
		NumSubscriptions:  ablyNumSubscriptions(),
		MessageDataLength: ablyMessageDataLength(),
		Publisher:         ablyPublisher(),
		NumChannels:       ablyNumChannels(),
	}
}

func getEnv(name string) string {
	value, exists := os.LookupEnv(name)

	if !exists {
		panic("Environment Variable '" + name + "' not set!")
	}

	return value
}

func getEnvWithDefault(name string, defaultValue string) string {
	value, exists := os.LookupEnv(name)

	if exists {
		return value
	}

	return defaultValue
}

func ablyPublisher() bool {
	return getEnvWithDefault("ABLY_PUBLISHER", DefaultPublisher) == "true"
}

func ablyNumChannels() int {
	value := getEnvWithDefault("ABLY_NUM_CHANNELS", DefaultNumChannels)

	n, err := strconv.Atoi(value)

	if err != nil {
		panic("Expected an Integer for 'ABLY_NUM_CHANNELS' - got '" + value + "'")
	}

	return n
}

func ablyTestType() string {
	return strings.ToLower(getEnv("ABLY_TEST_TYPE"))
}

func ablyEnv() string {
	return getEnv("ABLY_ENV")
}

func ablyAPIKey() string {
	return getEnv("ABLY_API_KEY")
}

func ablyChannelName() string {
	return getEnvWithDefault("ABLY_CHANNEL_NAME", DefaultChannelName)
}

func ablyPublishInterval() int {
	value := getEnvWithDefault("ABLY_PUBLISH_INTERVAL", DefaultPublishInterval)

	n, err := strconv.Atoi(value)

	if err != nil {
		panic("Expected an Integer for 'ABLY_PUBLISH_INTERVAL' - got '" + value + "'")
	}

	return n
}

func ablyNumSubscriptions() int {
	value := getEnvWithDefault("ABLY_NUM_SUBSCRIPTIONS", DefaultNumSubscriptions)

	n, err := strconv.Atoi(value)

	if err != nil {
		panic("Expected an Integer for 'ABLY_NUM_SUBSCRIPTIONS' - got '" + value + "'")
	}

	return n
}

func ablyMessageDataLength() int {
	value := getEnvWithDefault("ABLY_MSG_DATA_LENGTH", DefaultMessageDataLength)

	n, err := strconv.Atoi(value)

	if err != nil {
		panic("Expected an Integer for 'ABLY_MSG_DATA_LENGTH' - got '" + value + "'")
	}

	return n
}
