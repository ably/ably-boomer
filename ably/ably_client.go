package main

import (
	"github.com/ably/ably-go/ably"
)

// newAblyClient creates a new Ably realtime client.
func newAblyClient(testConfig TestConfig) (*ably.Realtime, error) {
	options := ably.ClientOptions{}.
		Key(testConfig.APIKey).
		Environment(testConfig.Env)

	return ably.NewRealtime(options)
}
