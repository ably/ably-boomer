package ably

import (
	"context"
	"fmt"

	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-boomer/tasks"
	"github.com/ably/ably-go/ably"
	"github.com/ably/ably-go/ably/proto"
	"github.com/inconshreveable/log15"
)

// TaskFn creates an Ably task to run with boomer.
func TaskFn(log log15.Logger, conf config.Conf) (func(), error) {
	f, err := newFactory(conf)
	if err != nil {
		return nil, err
	}
	return func() {
		defer f.Close()

		var sf tasks.SubscriberFactory = f
		if conf.SSESubscriber {
			sf = newSSESubscriberFactory(conf)
		}

		tasks.NewTask(log, conf, sf, f).Run()
	}, nil
}

type factory struct {
	*ably.RealtimeClient
}

func newFactory(conf config.Conf) (*factory, error) {
	c, err := newAblyClient(conf.APIKey, conf.Env)
	if err != nil {
		return nil, err
	}
	return &factory{RealtimeClient: c}, nil
}

func (f *factory) NewPublisher(ctx context.Context, channelName string) (tasks.Publisher, error) {
	p := &pubSub{c: f.Channels.Get(channelName)}
	r, err := p.c.Attach()
	if err != nil {
		return nil, fmt.Errorf("attaching to %s: %w", channelName, err)
	}
	if err := r.Wait(); err != nil {
		return nil, fmt.Errorf("waiting to attach to %s: %w", channelName, err)
	}
	return p, nil
}

func (f *factory) NewSubscriber(ctx context.Context, channelName string) (tasks.Subscriber, error) {
	return &pubSub{c: f.Channels.Get(channelName)}, nil
}

type pubSub struct {
	c *ably.RealtimeChannel
}

func (ps *pubSub) Publish(_ context.Context, msg *proto.Message) error {
	r, err := ps.c.PublishAll([]*proto.Message{msg})
	if err != nil {
		return fmt.Errorf("publishing: %w", err)
	}
	if err := r.Wait(); err != nil {
		return fmt.Errorf("waiting for ack: %w", err)
	}
	return nil
}

func (ps *pubSub) Subscribe(ctx context.Context, msgHandler func(msg *proto.Message)) error {
	sub, err := ps.c.Subscribe()
	if err != nil {
		return fmt.Errorf("subscribing: %w", err)
	}
	defer sub.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-sub.MessageChannel():
			msgHandler(msg)
		}
	}
}

func newAblyClient(apiKey, env string) (*ably.RealtimeClient, error) {
	options := ably.NewClientOptions(apiKey)
	options.Environment = env

	return ably.NewRealtimeClient(options)
}
