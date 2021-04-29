package ablyboomer

import (
	"context"
	"testing"
	"time"

	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-go/ably"
	"github.com/inconshreveable/log15"
	"github.com/myzhan/boomer"
)

// TestWorkerStandalone tests running a standalone Worker with subscriber,
// publisher and presence tasks enabled.
func TestWorkerStandalone(t *testing.T) {
	// initialise the worker to run standalone with a test client
	conf := config.Default()
	conf.Client = randomString(16)
	conf.Standalone.Enabled = true
	conf.Standalone.Users = 2
	conf.Standalone.SpawnRate = 2
	conf.Subscriber.Enabled = true
	conf.Subscriber.Channels = "test-{{ mod .UserNumber 2 }}-1,test-{{ .UserNumber }}-2"
	conf.Publisher.Enabled = true
	conf.Publisher.Channels = "test-{{ mod .UserNumber 2 }}-1,test-{{ .UserNumber }}-2"
	conf.Publisher.PublishInterval = time.Second
	conf.Presence.Enabled = true
	conf.Presence.Channels = "test-{{ mod .UserNumber 2 }}-1"
	conf.Log.Level = "debug"

	events := make(chan testEvent, 12)
	RegisterNewClientFunc(conf.Client, newTestClientFunc(events))
	worker, err := NewWorker(conf)
	if err != nil {
		t.Fatal(err)
	}

	// run the worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Run(ctx)

	// define a convenience function to wait for events
	waitEvents := func(done func(testEvent) bool) {
		timeout := time.After(10 * time.Second)
		for {
			select {
			case event := <-events:
				if done(event) {
					return
				}
			case <-timeout:
				t.Fatal("timed out waiting for test events")
			}
		}
	}

	// wait for the worker to spawn 4 subscribers, publish 4 messages and enter 2 channels
	subscribes := 0
	publishes := 0
	presences := 0
	waitEvents(func(event testEvent) bool {
		switch event {
		case testEventPublish:
			publishes++
		case testEventSubscribe:
			subscribes++
		case testEventPresence:
			presences++
		}
		return subscribes == 4 && publishes == 4 && presences == 2
	})

	// stop the load test, check the users all stop
	boomer.Events.Publish("boomer:stop")
	stopped := 0
	waitEvents(func(event testEvent) bool {
		if event != testEventStop {
			t.Fatalf("expected stop event, got %v", event)
		}
		stopped++
		return stopped == 2
	})
}

func newTestClientFunc(events chan testEvent) NewClientFunc {
	return func(ctx context.Context, conf *config.Config, log log15.Logger) (Client, error) {
		return &testClient{events}, nil
	}
}

type testClient struct {
	events chan testEvent
}

func (t *testClient) Subscribe(ctx context.Context, channel string, handler func(*ably.Message)) error {
	t.events <- testEventSubscribe
	<-ctx.Done()
	t.events <- testEventStop
	return ctx.Err()
}

func (t *testClient) Publish(ctx context.Context, channel string, messages []*ably.Message) error {
	select {
	case t.events <- testEventPublish:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *testClient) Enter(ctx context.Context, channel, clientID string) error {
	select {
	case t.events <- testEventPresence:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *testClient) Close() error {
	return nil
}

type testEvent string

const (
	testEventPublish   testEvent = "publish"
	testEventSubscribe testEvent = "subscribe"
	testEventPresence  testEvent = "presence"
	testEventStop      testEvent = "stop"
)
