package main

import (
	"time"

	"github.com/ably/ably-go/ably"
)

// newAblyClient creates a new Ably realtime client, making multiple attempts
// to avoid transient connection errors being reported as failures.
func newAblyClient(testConfig TestConfig) (client *ably.RealtimeClient, err error) {
	options := ably.NewClientOptions(testConfig.APIKey)
	options.Environment = testConfig.Env

	timeout := 30 * time.Second
	delay := 100 * time.Millisecond
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(delay) {
		client, err = ably.NewRealtimeClient(options)
		if err == nil {
			return
		}
		log.Warn("error creating ably client in retry loop", "err", err)
	}
	return
}
