package ably

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-go/ably"
	"github.com/inconshreveable/log15"
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

	subClients := []ably.Realtime{}

	log.Info("creating subscribers", "channel", channelName, "count", config.NumSubscriptions)
	for i := 0; i < config.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			log.Info("creating subscriber realtime connection", "num", i+1)
			subClient, err := newAblyClient(config, log)
			if err != nil {
				log.Error("error creating subscriber realtime connection", "num", i+1, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer subClient.Close()

			subClients = append(subClients, *subClient)

			channel := subClient.Channels.Get(channelName)

			log.Info("creating subscriber", "num", i+1, "channel", channelName)
			sub, err := channel.Subscribe()
			if err != nil {
				log.Error("error creating subscriber", "num", i+1, "channel", channelName, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer sub.Close()

			wg.Add(1)
			go reportSubscriptionToLocust(ctx, sub, subClient.Connection, errorChannel, &wg, log.New("channel", channelName))
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
			err := publishWithRetries(channel, timePublished, data, log)

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
func publishWithRetries(channel *ably.RealtimeChannel, name string, data interface{}, log log15.Logger) (err error) {
	timeout := 30 * time.Second
	delay := 100 * time.Millisecond
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(delay) {
		_, err = channel.Publish(name, data)
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
