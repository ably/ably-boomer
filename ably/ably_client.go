package ably

import (
	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-go/ably"
	"github.com/inconshreveable/log15"
)

// newAblyClient creates a new Ably realtime client and waits for it to connect.
func newAblyClient(config *config.Config, log log15.Logger) (*ably.Realtime, error) {
	options := ably.ClientOptions{}.
		Key(config.APIKey).
		Environment(config.Env).
		AutoConnect(false)

	client, err := ably.NewRealtime(options)
	if err != nil {
		return nil, err
	}

	// wait for either a CONNECTED or FAILED event
	errC := make(chan error)
	unsub := client.Connection.OnAll(func(state ably.ConnectionStateChange) {
		log.Info("got connection state change", "event", state.Event, "reason", state.Reason)
		switch state.Event {
		case ably.ConnectionEventConnected:
			errC <- nil
		case ably.ConnectionEventFailed:
			errC <- state.Reason
		}
	})
	defer unsub()
	client.Connect()
	if err := <-errC; err != nil {
		return nil, err
	}
	return client, nil
}
