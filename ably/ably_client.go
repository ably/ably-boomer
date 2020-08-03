package main

import (
	"github.com/ably/ably-go/ably"
)

func newAblyClient(testConfig TestConfig) (*ably.RealtimeClient, error) {
	options := ably.NewClientOptions(testConfig.APIKey)
	options.Environment = testConfig.Env

	return ably.NewRealtimeClient(options)
}
