package ablyboomer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-go/ably"
	"github.com/inconshreveable/log15"
	"github.com/myzhan/boomer"
	"github.com/r3labs/sse"
)

// Client is a client that a Locust user instantiates and uses to subscribe,
// publish and be present on channels.
type Client interface {
	// Subscribe subscribes to the given channel and calls the given
	// handler for each message received.
	Subscribe(ctx context.Context, channel string, handler func(msg *ably.Message)) error

	// Publish publishes the given message on the given channel.
	Publish(ctx context.Context, channel string, messages []*ably.Message) error

	// Enter enters the given channel using the given clientID.
	Enter(ctx context.Context, channel, clientID string) error

	// Close closes the client.
	Close() error
}

// NewClientFunc is the type of function that initialises a client, and is
// typically NewAblyClient but may also be a custom function if ablyboomer
// is used as a library to test using different types of clients.
type NewClientFunc func(context.Context, *config.Config, log15.Logger) (Client, error)

// newClientFuncs is the list of registered NewClientFuncs
var newClientFuncs = make(map[string]NewClientFunc)

// RegisterNewClientFunc registers a NewClientFunc with the given name that can
// be referenced as the client to use using config.Client.
func RegisterNewClientFunc(name string, f NewClientFunc) {
	if _, ok := newClientFuncs[name]; ok {
		panic(fmt.Sprintf("client already registered: %s", name))
	}
	newClientFuncs[name] = f
}

// GetNewClientFunc gets the registered NewClientFunc with the given name.
func GetNewClientFunc(name string) (NewClientFunc, bool) {
	f, ok := newClientFuncs[name]
	return f, ok
}

func init() {
	// register the ably and ably-sse NewClientFuncs
	RegisterNewClientFunc("ably", NewAblyClient)
	RegisterNewClientFunc("ably-sse", NewAblySSEClient)
}

// NewAblyClient is a NewClientFunc that initialises an Ably realtime client.
//
// A goroutine is started to watch and report connection events, and the client
// is only returned once a CONNECTED event is received.
func NewAblyClient(ctx context.Context, conf *config.Config, log log15.Logger) (Client, error) {
	client, err := ably.NewRealtime(conf.Ably.ClientOptions()...)
	if err != nil {
		return nil, err
	}

	// watch connection events
	firstErr := make(chan error)
	go func() {
		var once sync.Once
		var disconnectedAt int64
		done := make(chan struct{})
		unsub := client.Connection.OnAll(func(state ably.ConnectionStateChange) {
			log.Debug("got ably connection state change", "event", state.Event, "reason", state.Reason)
			switch state.Event {
			case ably.ConnectionEventConnected:
				once.Do(func() { firstErr <- nil })
				if disconnectedAt > 0 {
					reconnectLatency := timeNow() - disconnectedAt
					boomer.RecordSuccess("ablyboomer", "reconnect", reconnectLatency, 0)
				}
			case ably.ConnectionEventDisconnected:
				disconnectedAt = timeNow()
			case ably.ConnectionEventFailed:
				once.Do(func() { firstErr <- state.Reason })
			case ably.ConnectionEventClosed:
				close(done)
			}
		})
		defer unsub()
		<-done
	}()

	// connect and wait for the first CONNECTED or FAILED event
	client.Connect()
	select {
	case err := <-firstErr:
		return &ablyClient{client}, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ablyClient implements the Client interface using a wrapped ably.Realtime
// client.
type ablyClient struct {
	*ably.Realtime
}

// Subscribe subscribes to the given Ably channel and calls the given handler
// with the data of each message received.
func (a *ablyClient) Subscribe(ctx context.Context, channelName string, handler func(*ably.Message)) error {
	channel := a.Realtime.Channels.Get(channelName)
	unsub, err := channel.SubscribeAll(ctx, func(msg *ably.Message) {
		handler(msg)
	})
	if err != nil {
		return err
	}
	defer unsub()
	<-ctx.Done()
	return ctx.Err()
}

// Publish publishes the given message to the given Ably channel.
func (a *ablyClient) Publish(ctx context.Context, channelName string, messages []*ably.Message) error {
	return a.Realtime.Channels.Get(channelName).PublishBatch(ctx, messages)
}

// Enter enters the given Ably channel using the given clientID.
func (a *ablyClient) Enter(ctx context.Context, channelName, clientID string) error {
	return a.Realtime.Channels.Get(channelName).Presence.EnterClient(ctx, clientID, "")
}

// Close closes the underlying ably.Realtime client.
func (a *ablyClient) Close() error {
	a.Realtime.Close()
	return nil
}

// NewAblySSEClient is a NewClientFunc that initialises a client that
// subscribes to Ably channels using Server-Sent-Events (SSE).
func NewAblySSEClient(ctx context.Context, conf *config.Config, log log15.Logger) (Client, error) {
	return &ablySSEClient{
		conf: conf,
		stop: make(chan struct{}),
	}, nil
}

// ablySSEClient implements the Subscribe method of the Client interface using
// the github.com/r3labs/sse library.
type ablySSEClient struct {
	conf     *config.Config
	stopOnce sync.Once
	stop     chan struct{}
}

// Subscribe subscribes to the given Ably channel using SSE and calls the given
// handler with the data of each non-empty message received.
func (a *ablySSEClient) Subscribe(ctx context.Context, channelName string, handler func(*ably.Message)) error {
	u := url.URL{
		Scheme:   "https",
		Host:     "realtime.ably.io",
		Path:     "/sse",
		RawQuery: "channels=" + channelName + "&v=1.1&key=" + a.conf.Ably.APIKey,
	}
	if env := a.conf.Ably.Environment; env != "" && env != "production" {
		u.Host = env + "-" + u.Host
	}
	client := sse.NewClient(u.String())
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-a.stop
		cancel()
	}()
	return client.SubscribeWithContext(ctx, "", func(event *sse.Event) {
		if len(event.Data) > 0 {
			var msg ably.Message
			json.Unmarshal(event.Data, &msg)
			handler(&msg)
		}
	})
}

// Publish is not compatible with SSE.
func (a *ablySSEClient) Publish(ctx context.Context, channelName string, messages []*ably.Message) error {
	return errors.New("Publish not implemented for SSE client")
}

// Enter is not compatible with SSE.
func (a *ablySSEClient) Enter(ctx context.Context, channelName, clientID string) error {
	return errors.New("Enter not implemented for SSE client")
}

// Close stops any active subscriptions.
func (a *ablySSEClient) Close() error {
	a.stopOnce.Do(func() { close(a.stop) })
	return nil
}
