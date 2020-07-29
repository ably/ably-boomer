package main

import (
	"os"
	"strconv"
	"strings"
)

const DefaultChannelName = "test_channel"
const DefaultCPUProfile = ""
const DefaultPublishInterval = "10"
const DefaultNumSubscriptions = "2"
const DefaultMessageDataLength = "2000"
const DefaultPublisher = "false"
const DefaultNumChannels = "64"

type TestConfig struct {
	TestType          string
	Env               string
	ApiKey            string
	ChannelName       string
	CPUProfile        string
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
		ApiKey:            ablyApiKey(),
		ChannelName:       ablyChannelName(),
		CPUProfile:        ablyCPUProfile(),
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

func ablyApiKey() string {
	return getEnv("ABLY_API_KEY")
}

func ablyChannelName() string {
	return getEnvWithDefault("ABLY_CHANNEL_NAME", DefaultChannelName)
}

func ablyCPUProfile() string {
	return getEnvWithDefault("ABLY_CPU_PROFILE", DefaultCPUProfile)
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
