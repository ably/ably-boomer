package ably

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-boomer/tasks"
	"github.com/ably/ably-go/ably/proto"
	"github.com/r3labs/sse"
)

type wrappedSSEClient struct {
	*sse.Client
}

func (wc *wrappedSSEClient) Subscribe(ctx context.Context, msgHandler func(message *proto.Message)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var innerErr error
	outerErr := wc.SubscribeWithContext(ctx, "", func(msg *sse.Event) {
		m := &proto.Message{}
		if innerErr = m.UnmarshalJSON(msg.Data); innerErr != nil {
			cancel()
			return
		}
		msgHandler(m)
	})
	if innerErr != nil {
		return fmt.Errorf("unmarshalling sse event: %w", innerErr)
	}
	if outerErr != nil && ctx.Err() == nil {
		return outerErr
	}
	return nil
}

type sseSubscriberFactory struct {
	conf config.Conf
}

func newSSESubscriberFactory(conf config.Conf) *sseSubscriberFactory {
	return &sseSubscriberFactory{conf}
}

func (s *sseSubscriberFactory) NewSubscriber(ctx context.Context, channelName string) (tasks.Subscriber, error) {
	url := url.URL{
		Scheme:   "https",
		Host:     "realtime.ably.io",
		Path:     "/sse",
		RawQuery: "channels=" + channelName + "&v=1.1&key=" + s.conf.APIKey,
	}
	if s.conf.Env != "" && s.conf.Env != "production" {
		url.Host = s.conf.Env + "-" + url.Host
	}
	return &wrappedSSEClient{Client: sse.NewClient(url.String())}, nil
}
