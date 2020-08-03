package main

import (
	"os"
	"strconv"
	"testing"
)

func unsetEnv() {
	os.Unsetenv("ABLY_TEST_TYPE")
	os.Unsetenv("ABLY_ENV")
	os.Unsetenv("ABLY_API_KEY")
	os.Unsetenv("ABLY_CHANNEL_NAME")
	os.Unsetenv("ABLY_PUBLISH_INTERVAL")
	os.Unsetenv("ABLY_NUM_SUBSCRIPTIONS")
	os.Unsetenv("ABLY_MSG_DATA_LENGTH")
}

func assertPanic(t *testing.T, f func() TestConfig) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected a panic, didn't get one.")
		}
	}()
	f()
}

func TestNewTestConfig(t *testing.T) {
	t.Run("required environment variables are not set", func(ts *testing.T) {
		ts.Run("'ABLY_TEST_TYPE' not set", func(ts *testing.T) {
			os.Setenv("ABLY_ENV", "env")
			os.Setenv("ABLY_API_KEY", "apiKey")
			defer unsetEnv()

			assertPanic(t, newTestConfig)
		})

		ts.Run("'ABLY_ENV' not set", func(ts *testing.T) {
			os.Setenv("ABLY_TEST_TYPE", "testType")
			os.Setenv("ABLY_API_KEY", "apiKey")
			defer unsetEnv()

			assertPanic(t, newTestConfig)
		})

		ts.Run("'ABLY_API_KEY' not set", func(ts *testing.T) {
			os.Setenv("ABLY_TEST_TYPE", "testType")
			os.Setenv("ABLY_ENV", "env")
			defer unsetEnv()

			assertPanic(t, newTestConfig)
		})
	})

	t.Run("only required environment variables set", func(ts *testing.T) {
		testType := "personal"
		env := "loadtest"
		apiKey := "key"

		defaultChannelName := "test_channel"
		defaultPublishInterval := 10
		defaultNumSubscriptions := 2
		defaultMessageDataLength := 2000

		os.Setenv("ABLY_TEST_TYPE", testType)
		os.Setenv("ABLY_ENV", env)
		os.Setenv("ABLY_API_KEY", apiKey)
		defer unsetEnv()

		testConfig := newTestConfig()

		if testConfig.TestType != testType {
			t.Errorf("TestType was incorrect, got: %s, wanted: %s.", testConfig.TestType, testType)
		}

		if testConfig.Env != env {
			t.Errorf("Env was incorrect, got: %s, wanted: %s.", testConfig.Env, env)
		}

		if testConfig.APIKey != apiKey {
			t.Errorf("ApiKey was incorrect, got: %s, wanted: %s.", testConfig.APIKey, apiKey)
		}

		if testConfig.ChannelName != defaultChannelName {
			t.Errorf("ChannelName was incorrect, got: %s, wanted: %s.", testConfig.ChannelName, defaultChannelName)
		}

		if testConfig.PublishInterval != defaultPublishInterval {
			t.Errorf("PublishInterval was incorrect, got: %d, wanted: %d.", testConfig.PublishInterval, defaultPublishInterval)
		}

		if testConfig.NumSubscriptions != defaultNumSubscriptions {
			t.Errorf("NumSubscriptions was incorrect, got: %d, wanted: %d.", testConfig.NumSubscriptions, defaultNumSubscriptions)
		}

		if testConfig.MessageDataLength != defaultMessageDataLength {
			t.Errorf("MessageDataLength was incorrect, got: %d, wanted: %d.", testConfig.MessageDataLength, defaultMessageDataLength)
		}
	})

	t.Run("all environment variables set", func(ts *testing.T) {
		testType := "personal"
		env := "loadtest"
		apiKey := "key"
		channelName := "different-ably-channel"
		publishInterval := 60
		numSubscriptions := 31250
		messageDataLength := 9001

		os.Setenv("ABLY_TEST_TYPE", testType)
		os.Setenv("ABLY_ENV", env)
		os.Setenv("ABLY_API_KEY", apiKey)
		os.Setenv("ABLY_CHANNEL_NAME", channelName)
		os.Setenv("ABLY_PUBLISH_INTERVAL", strconv.Itoa(publishInterval))
		os.Setenv("ABLY_NUM_SUBSCRIPTIONS", strconv.Itoa(numSubscriptions))
		os.Setenv("ABLY_MSG_DATA_LENGTH", strconv.Itoa(messageDataLength))
		defer unsetEnv()

		testConfig := newTestConfig()

		if testConfig.TestType != testType {
			t.Errorf("TestType was incorrect, got: %s, wanted: %s.", testConfig.TestType, testType)
		}

		if testConfig.Env != env {
			t.Errorf("Env was incorrect, got: %s, wanted: %s.", testConfig.Env, env)
		}

		if testConfig.APIKey != apiKey {
			t.Errorf("ApiKey was incorrect, got: %s, wanted: %s.", testConfig.APIKey, apiKey)
		}

		if testConfig.ChannelName != channelName {
			t.Errorf("ChannelName was incorrect, got: %s, wanted: %s.", testConfig.ChannelName, channelName)
		}

		if testConfig.PublishInterval != publishInterval {
			t.Errorf("PublishInterval was incorrect, got: %d, wanted: %d.", testConfig.PublishInterval, publishInterval)
		}

		if testConfig.NumSubscriptions != numSubscriptions {
			t.Errorf("NumSubscriptions was incorrect, got: %d, wanted: %d.", testConfig.NumSubscriptions, numSubscriptions)
		}

		if testConfig.MessageDataLength != messageDataLength {
			t.Errorf("MessageDataLength was incorrect, got: %d, wanted: %d.", testConfig.MessageDataLength, messageDataLength)
		}
	})
}
