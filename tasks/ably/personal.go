package ably

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
	ablyrpc "github.com/ably/ably-go/ably/proto"
	"github.com/inconshreveable/log15"
	"github.com/r3labs/sse"
)

// PersonalConf is the Personal task's configuration.
type PersonalConf struct {
	Logger           log15.Logger
	APIKey           string
	Env              string
	PublishInterval  int
	NumSubscriptions int
	MsgDataLength    int
	SSESubscriber    bool
}

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

func personalTask(conf PersonalConf) {
	log := conf.Logger
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	var wg sync.WaitGroup
	errorChannel := make(chan error)

	channelName := randomString(channelNameLength)

	var subClients []io.Closer
	log.Info("creating subscribers", "channel", channelName, "count", conf.NumSubscriptions)
	if conf.SSESubscriber {
		var err error
		if subClients, err = createSSESubscribers(ctx, conf, log, channelName, wg, errorChannel); err != nil {
			log.Error("creating sse subscribers", "err", err)
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}
	} else {
		var err error
		if subClients, err = createAblySubscribers(ctx, conf, log, channelName, wg, errorChannel); err != nil {
			log.Error("creating subscribers", "err", err)
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}
	}

	log.Info("creating publisher realtime connection")
	publishClient, err := newAblyClient(conf.APIKey, conf.Env)
	if err != nil {
		log.Error("error creating publisher realtime connection", "err", err)
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		return
	}
	defer publishClient.Close()

	channel := publishClient.Channels.Get(channelName)
	defer channel.Close()

	cleanup := func() {
		publishClient.Close()

		for _, subClient := range subClients {
			subClient.Close()
		}
	}

	randomDelay(log)

	log.Info("creating publisher", "channel", channelName, "period", conf.PublishInterval)
	ticker := time.NewTicker(time.Duration(conf.PublishInterval) * time.Second)
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
			data := randomString(conf.MsgDataLength)
			timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

			log.Info("publishing message", "size", len(data))
			_, err := channel.Publish(timePublished, data)

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

type wrappedSSEClient struct {
	*sse.Client
	cancel context.CancelFunc
}

func (w wrappedSSEClient) Close() error {
	w.cancel()
	return nil
}

func createSSESubscribers(ctx context.Context, conf PersonalConf, log log15.Logger, channelName string, wg sync.WaitGroup, errorChannel chan error) ([]io.Closer, error) {
	subClients := make([]io.Closer, 0, conf.NumSubscriptions)
	for i := 0; i < conf.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			ctx, cancel := context.WithCancel(ctx)
			log.Info("creating subscriber sse connection", "num", i+1)
			url := url.URL{
				Scheme:   "https",
				Host:     ably.RestHost,
				Path:     "/sse",
				RawQuery: "channels=" + channelName + "&v=1.2&key=" + conf.APIKey,
			}
			if conf.Env != "" {
				url.Host = conf.Env + "-" + url.Host
			}
			subClient := sse.NewClient(url.String())
			subClients = append(subClients, wrappedSSEClient{Client: subClient, cancel: cancel})

			wg.Add(1)
			go func() {
				if err := subClient.SubscribeWithContext(ctx, "", func(msg *sse.Event) {
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
					log.Error("subscribing", "err", err)
					boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				}
			}()
		}
	}
	return subClients, nil
}

func createAblySubscribers(ctx context.Context, conf PersonalConf, log log15.Logger, channelName string, wg sync.WaitGroup, errorChannel chan error) ([]io.Closer, error) {
	subClients := make([]io.Closer, 0, conf.NumSubscriptions)
	for i := 0; i < conf.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			log.Info("creating subscriber realtime connection", "num", i+1)
			subClient, err := newAblyClient(conf.APIKey, conf.Env)
			if err != nil {
				subClient.Close()
				return nil, fmt.Errorf("creating  %dth subscriber; %w", i+1, err)
			}

			channel := subClient.Channels.Get(channelName)

			log.Info("creating subscriber", "num", i+1, "channel", channelName)
			sub, err := channel.Subscribe()
			if err != nil {
				subClient.Close()
				return nil, fmt.Errorf("subscribing to %dth channel; %w", i+1, err)
			}

			wg.Add(1)
			go reportSubscriptionToLocust(ctx, sub, subClient.Connection, errorChannel, &wg, log.New("channel", channelName))
			subClients = append(subClients, subClient)
		}
	}
	return subClients, nil
}

// CurryPersonalTask returns a function allowing to run the Personal task.
func CurryPersonalTask(conf PersonalConf) func() {
	log := conf.Logger
	log.Info(
		"starting personal task",
		"env", conf.Env,
		"publish-interval", conf.PublishInterval,
		"subs-per-channel", conf.NumSubscriptions,
		"message-size", conf.MsgDataLength,
	)

	return func() {
		personalTask(conf)
	}
}
