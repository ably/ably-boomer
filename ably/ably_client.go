package main

import (
	"github.com/ably/ably-go/ably"
)

// newAblyClient creates a new Ably realtime client.
func newAblyClient(testConfig TestConfig) (*ably.Realtime, error) {
	options := ably.NewClientOptions(testConfig.APIKey)
	options.Environment(testConfig.Env)

	return ably.NewRealtime(options)
}
