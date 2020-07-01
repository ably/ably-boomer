package main

import (
	"log"
	"os"
	"strings"
)

const DefaultChannelName = "test_channel"

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
