package main

import (
	"github.com/ably/ably-go/ably"
)

func newAblyClient(testConfig TestConfig) *ably.RealtimeClient {
	options := ably.NewClientOptions(testConfig.ApiKey)
	options.Environment = testConfig.Env

	client, err := ably.NewRealtimeClient(options)
	if err != nil {
		panic(err.Error())
	}

	return client
}
