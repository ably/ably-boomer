package main

import (
	"github.com/ably/ably-go/ably"
)

func newAblyClient(testConfig TestConfig) (*ably.RealtimeClient, error) {
	options := ably.NewClientOptions(testConfig.ApiKey)
	options.Environment = testConfig.Env

	client, err := ably.NewRealtimeClient(options)
	if err != nil {
		return nil, err
	}

	return client, nil
}
