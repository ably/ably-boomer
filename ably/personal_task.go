package ably

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-go/ably"
	ablyrpc "github.com/ably/ably-go/ably/proto"
	"github.com/inconshreveable/log15"
	"github.com/r3labs/sse"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

const channelNameLength = 100

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randomDelay(log log15.Logger) {
	r := rand.Intn(60)
	log.Info("introducing random delay", "seconds", r)
	time.Sleep(time.Duration(r) * time.Second)
	log.Info("continuing after random delay")
}

func personalTask(config *config.Config, log log15.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	var wg sync.WaitGroup
	errorChannel := make(chan error)

	channelName := randomString(channelNameLength)

	var subClients []closer

	log.Info("creating subscribers", "channel", channelName, "count", config.NumSubscriptions)
	if config.SSESubscriber {
		var err error
		if subClients, err = createSSESubscribers(ctx, config, log, channelName, &wg); err != nil {
			log.Error("creating sse subscribers", "err", err)
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}
	} else {
		var err error
		if subClients, err = createAblySubscribers(ctx, config, log, channelName, &wg, errorChannel); err != nil {
			log.Error("creating subscribers", "err", err)
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}
	}

	log.Info("creating publisher realtime connection")
	publishClient, err := newAblyClient(config, log)
	if err != nil {
		log.Error("error creating publisher realtime connection", "err", err)
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		return
	}
	defer publishClient.Close()

	channel := publishClient.Channels.Get(channelName)

	cleanup := func() {
		publishClient.Close()

		for _, subClient := range subClients {
			subClient.Close()
		}
	}

	randomDelay(log)

	log.Info("creating publisher", "channel", channelName, "period", config.PublishInterval)
	ticker := time.NewTicker(time.Duration(config.PublishInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-errorChannel:
			log.Error("error from subscriber goroutine", "err", err)
			cancel()
			wg.Wait()
			cleanup()
			return
		case <-ticker.C:
			data := randomString(config.MessageDataLength)
			timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

			log.Info("publishing message", "size", len(data))
			err := publishWithRetries(ctx, channel, timePublished, data, log)

			if err != nil {
				log.Error("error publishing message", "err", err)
				boomer.RecordFailure("ably", "publish", 0, err.Error())
			} else {
				boomer.RecordSuccess("ably", "publish", 0, 0)
			}
		case <-ctx.Done():
			log.Info("personal task context done, cleaning up")
			wg.Wait()
			cleanup()
			return
		}
	}
}

// publishWithRetries makes multiple attempts to publish a message on the given
// channel.
//
// TODO: remove the retries once handled by ably-go.
func publishWithRetries(ctx context.Context, channel *ably.RealtimeChannel, name string, data interface{}, log log15.Logger) (err error) {
	timeout := 30 * time.Second
	delay := 100 * time.Millisecond
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(delay) {
		err = channel.Publish(ctx, name, data)
		if err == nil || !isRetriablePublishError(err) {
			return
		}
		log.Warn("error publishing message in retry loop", "err", err)
	}
	return
}

// isRetriablePublishError returns whether the given publish error is retriable
// by checking if the error string indicates the connection is in a failed
// state.
//
// TODO: remove once this is handled by ably-go.
func isRetriablePublishError(err error) bool {
	return strings.Contains(err.Error(), "attempted to attach channel to inactive connection")
}

type closer interface {
	Close()
}

type wrappedSSEClient struct {
	*sse.Client
	cancel context.CancelFunc
}

func (w wrappedSSEClient) Close() {
	w.cancel()
}

func createSSESubscribers(ctx context.Context, conf *config.Config, log log15.Logger, channelName string, wg *sync.WaitGroup) ([]closer, error) {
	subClients := make([]closer, 0, conf.NumSubscriptions)
	for i := 0; i < conf.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			i := i
			ctx, cancel := context.WithCancel(ctx)
			url := url.URL{
				Scheme:   "https",
				Host:     "realtime.ably.io",
				Path:     "/sse",
				RawQuery: "channels=" + channelName + "&v=1.1&key=" + conf.APIKey,
			}
			if conf.Env != "" && conf.Env != "production" {
				url.Host = conf.Env + "-" + url.Host
			}
			log.Info("creating subscriber sse connection", "num", i+1, "url", url)
			subClient := sse.NewClient(url.String())
			subClients = append(subClients, wrappedSSEClient{Client: subClient, cancel: cancel})

			wg.Add(1)
			go func() {
				if err := subClient.SubscribeWithContext(ctx, "", func(msg *sse.Event) {
					if len(msg.Data) == 0 {
						// just ID message
						return
					}
					m := &ablyrpc.Message{}
					if err := m.UnmarshalJSON(msg.Data); err != nil {
						log.Error("unmarshalling message", "err", err)
						boomer.RecordFailure("ably", "subscribe", 0, err.Error())
					}
					validateMsg(m, log)
				}); err != nil {
					wg.Done()
					if ctx.Err() != nil {
						return
					}
					log.Error("subscribing", "err", err, "num", i+1)
					boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				}
			}()
		}
	}
	return subClients, nil
}

func createAblySubscribers(ctx context.Context, conf *config.Config, log log15.Logger, channelName string, wg *sync.WaitGroup, errorChannel chan error) ([]closer, error) {
	subClients := make([]closer, 0, conf.NumSubscriptions)
	for i := 0; i < conf.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			log.Info("creating subscriber realtime connection", "num", i+1)
			subClient, err := newAblyClient(conf, log)
			if err != nil {
				subClient.Close()
				return nil, fmt.Errorf("creating  %dth subscriber; %w", i+1, err)
			}

			channel := subClient.Channels.Get(channelName)

			log.Info("creating subscriber", "num", i+1, "channel", channelName)
			msgC := make(chan *ably.Message)
			_, err = channel.SubscribeAll(ctx, func(msg *ably.Message) {
				select {
				case msgC <- msg:
				case <-ctx.Done():
				}
			})
			if err != nil {
				subClient.Close()
				return nil, fmt.Errorf("subscribing to %dth channel; %w", i+1, err)
			}

			wg.Add(1)
			go reportSubscriptionToLocust(ctx, msgC, subClient.Connection, errorChannel, wg, log.New("channel", channelName))
			subClients = append(subClients, subClient)
		}
	}
	return subClients, nil
}

func curryPersonalTask(config *config.Config, log log15.Logger) func() {
	log.Info(
		"starting personal task",
		"env", config.Env,
		"publish-interval", config.PublishInterval,
		"subs-per-channel", config.NumSubscriptions,
		"message-size", config.MessageDataLength,
	)

	return func() {
		personalTask(config, log)
	}
}
