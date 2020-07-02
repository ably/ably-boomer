package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

const DefaultChannelName = "test_channel"
const DefaultPublishInterval = "10"
const DefaultNumSubscriptions = "2"

type TestConfig struct {
	TestType         string
	Env              string
	ApiKey           string
	ChannelName      string
	PublishInterval  int
	NumSubscriptions int
}

func newTestConfig() TestConfig {
	return TestConfig{
		TestType:         ablyTestType(),
		Env:              ablyEnv(),
		ApiKey:           ablyApiKey(),
		ChannelName:      ablyChannelName(),
		PublishInterval:  ablyPublishInterval(),
		NumSubscriptions: ablyNumSubscriptions(),
	}
}

func getEnv(name string) string {
	value, exists := os.LookupEnv(name)

	if !exists {
		log.Fatalln("Environment Variable '" + name + "' not set!")
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
