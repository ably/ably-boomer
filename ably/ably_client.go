package main

import (
	"time"

	"github.com/ably/ably-go/ably"
)

// newAblyClient creates a new Ably realtime client, making multiple attempts
// to avoid transient connection errors being reported as failures.
//
// TODO: remove the retries once handled by ably-go.
func newAblyClient(testConfig TestConfig) (client *ably.Realtime, err error) {
	options := ably.NewClientOptions(testConfig.APIKey)
	options.Environment(testConfig.Env)

	client, err = ably.NewRealtime(options)
	if err == nil {
		return
	}
	log.Warn("error creating client, will retry", "err", err)

	// retry with a backoff
	delays := []time.Duration{
		100 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
		10 * time.Second,
		10 * time.Second,
	}
	for _, delay := range delays {
		time.Sleep(delay)
		client, err = ably.NewRealtime(options)
		if err == nil {
			return
		}
		// ideally we'd only retry recoverable errors, but that really
		// ought to be done in ably-go, so let's defer to when that's
		// implemented in ably-go
		log.Warn("error creating ably client in retry loop", "err", err)
	}
	return
}
