package ably

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-go/ably"
	"github.com/inconshreveable/log15"
)

func generateChannelName(config *config.Config, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%config.NumChannels)
}

func publishOnInterval(ctx context.Context, config *config.Config, channel *ably.RealtimeChannel, delay int, errorChannel chan<- error, wg *sync.WaitGroup, log log15.Logger) {
	log = log.New("channel", channel.Name)
	log.Info("creating publisher", "period", config.PublishInterval)

	log.Info("introducing random delay before starting to publish", "milliseconds", delay)
	time.Sleep(time.Duration(delay) * time.Millisecond)
	log.Info("continuing after random delay")

	data := randomString(config.MessageDataLength)
	timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

	log.Info("publishing message", "size", len(data))
	if err := publishWithRetries(ctx, channel, timePublished, data, log); err != nil {
		log.Error("error publishing message", "err", err)
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		errorChannel <- err
		wg.Done()
		return
	} else {
		boomer.RecordSuccess("ably", "publish", 0, 0)
	}

	ticker := time.NewTicker(time.Duration(config.PublishInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data := randomString(config.MessageDataLength)
			timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

			log.Info("publishing message", "size", len(data), "millisecondTimestamp", millisecondTimestamp())
			if err := publishWithRetries(ctx, channel, timePublished, data, log); err != nil {
				log.Error("error publishing message", "err", err)
				boomer.RecordFailure("ably", "publish", 0, err.Error())
				errorChannel <- err
				ticker.Stop()
				wg.Done()
				return
			} else {
				boomer.RecordSuccess("ably", "publish", 0, 0)
			}
		case <-ctx.Done():
			ticker.Stop()
			wg.Done()
			return
		}
	}
}

func shardedPublisherTask(config *config.Config, log log15.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errorChannel := make(chan error)
	var wg sync.WaitGroup

	boomer.Events.Subscribe("boomer:stop", cancel)

	log.Info("creating realtime connection")
	client, err := newAblyClient(config, log)
	if err != nil {
		log.Error("error creating realtime connection", "err", err)
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		return
	}
	defer client.Close()

	log.Info("creating publishers", "count", config.NumChannels)
	for i := 0; i < config.NumChannels; i++ {
		channelName := generateChannelName(config, i)

		channel := client.Channels.Get(channelName)

		delay := i * config.PublishInterval / config.NumChannels

		log.Info("starting publisher", "num", i+1, "channel", channelName, "delay", delay)
		wg.Add(1)
		go publishOnInterval(ctx, config, channel, delay, errorChannel, &wg, log)
	}

	select {
	case err := <-errorChannel:
		log.Error("error from publisher goroutine", "err", err)
		cancel()
		client.Close()
		shardedPublisherTask(config, log)
	case <-ctx.Done():
		log.Info("sharded publisher task context done, cleaning up")
		cancel()
		client.Close()
		return
	}
}

func shardedSubscriberTask(config *config.Config, log log15.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	boomer.Events.Subscribe("boomer:stop", cancel)

	errorChannel := make(chan error)
	var wg sync.WaitGroup

	clients := []ably.Realtime{}

	log.Info("creating subscribers", "count", config.NumSubscriptions)
	for i := 0; i < config.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			log.Info("creating realtime connection", "num", i+1)
			client, err := newAblyClient(config, log)
			if err != nil {
				log.Error("error creating realtime connection", "num", i+1, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer client.Close()

			clients = append(clients, *client)

			channelName := generateChannelName(config, i)

			channel := client.Channels.Get(channelName)

			log.Info("creating subscriber", "num", i+1, "name", channelName)
			msgC := make(chan *ably.Message)
			unsub, err := channel.SubscribeAll(ctx, func(msg *ably.Message) {
				select {
				case msgC <- msg:
				case <-ctx.Done():
				}
			})
			if err != nil {
				log.Error("error creating subscriber", "num", i+1, "name", channelName, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer unsub()

			go reportSubscriptionToLocust(ctx, msgC, client.Connection, errorChannel, &wg, log.New("channel", channelName))
		}
	}

	cleanup := func() {
		for _, client := range clients {
			client.Close()
		}
	}

	for {
		select {
		case err := <-errorChannel:
			log.Error("error from subscriber goroutine", "err", err)
			cleanup()
			return
		case <-ctx.Done():
			log.Info("sharded subscriber task context done, cleaning up")
			cleanup()
			return
		}
	}
}

func curryShardedTask(config *config.Config, log log15.Logger) func() {
	if config.Publisher {
		log.Info(
			"starting sharded publisher task",
			"env", config.Env,
			"num-channels", config.NumChannels,
			"publish-interval", config.PublishInterval,
			"message-size", config.MessageDataLength,
		)
		return func() {
			shardedPublisherTask(config, log)
		}
	}

	log.Info(
		"starting sharded subscriber task",
		"env", config.Env,
		"num-channels", config.NumChannels,
		"subs-per-channel", config.NumSubscriptions,
	)
	return func() {
		shardedSubscriberTask(config, log)
	}
}
