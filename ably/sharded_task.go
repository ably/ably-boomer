package main

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
)

func generateChannelName(testConfig TestConfig, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%testConfig.NumChannels)
}

func publishOnInterval(ctx context.Context, testConfig TestConfig, channel *ably.RealtimeChannel, delay int, errorChannel chan<- error, wg *sync.WaitGroup) {
	log := log.New("channel", channel.Name)
	log.Info("creating publisher", "period", testConfig.PublishInterval)

	log.Info("introducing random delay before starting to publish", "seconds", delay)
	time.Sleep(time.Duration(delay) * time.Second)
	log.Info("continuing after random delay")

	data := randomString(testConfig.MessageDataLength)
	timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

	log.Info("publishing message", "size", len(data))
	if err := publishWithRetries(channel, timePublished, data); err != nil {
		log.Error("error publishing message", "err", err)
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		errorChannel <- err
		wg.Done()
		return
	} else {
		boomer.RecordSuccess("ably", "publish", 0, 0)
	}

	ticker := time.NewTicker(time.Duration(testConfig.PublishInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data := randomString(testConfig.MessageDataLength)
			timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

			log.Info("publishing message", "size", len(data))
			if err := publishWithRetries(channel, timePublished, data); err != nil {
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

func shardedPublisherTask(testConfig TestConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errorChannel := make(chan error)
	var wg sync.WaitGroup

	boomer.Events.Subscribe("boomer:stop", cancel)

	log.Info("creating realtime connection")
	client, err := newAblyClient(testConfig)
	if err != nil {
		log.Error("error creating realtime connection", "err", err)
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		return
	}
	defer client.Close()

	log.Info("creating publishers", "count", testConfig.NumChannels)
	for i := 0; i < testConfig.NumChannels; i++ {
		channelName := generateChannelName(testConfig, i)

		channel := client.Channels.Get(channelName)
		defer channel.Close()

		delay := i % testConfig.PublishInterval

		log.Info("starting publisher", "num", i+1, "channel", channelName, "delay", delay)
		wg.Add(1)
		go publishOnInterval(ctx, testConfig, channel, delay, errorChannel, &wg)
	}

	select {
	case err := <-errorChannel:
		log.Error("error from publisher goroutine", "err", err)
		cancel()
		client.Close()
		shardedPublisherTask(testConfig)
	case <-ctx.Done():
		log.Info("sharded publisher task context done, cleaning up")
		cancel()
		client.Close()
		return
	}
}

func shardedSubscriberTask(testConfig TestConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	boomer.Events.Subscribe("boomer:stop", cancel)

	errorChannel := make(chan error)
	var wg sync.WaitGroup

	clients := []ably.RealtimeClient{}

	log.Info("creating subscribers", "count", testConfig.NumSubscriptions)
	for i := 0; i < testConfig.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			log.Info("creating realtime connection", "num", i+1)
			client, err := newAblyClient(testConfig)
			if err != nil {
				log.Error("error creating realtime connection", "num", i+1, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer client.Close()

			clients = append(clients, *client)

			channelName := generateChannelName(testConfig, i)

			channel := client.Channels.Get(channelName)
			defer channel.Close()

			log.Info("creating subscriber", "num", i+1, "name", channelName)
			sub, err := channel.Subscribe()
			if err != nil {
				log.Error("error creating subscriber", "num", i+1, "name", channelName, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer sub.Close()

			go reportSubscriptionToLocust(ctx, sub, client.Connection, errorChannel, &wg, log.New("channel", channelName))
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

func curryShardedTask(testConfig TestConfig) func() {
	if testConfig.Publisher {
		log.Info(
			"starting sharded publisher task",
			"env", testConfig.Env,
			"num-channels", testConfig.NumChannels,
			"publish-interval", testConfig.PublishInterval,
			"message-size", testConfig.MessageDataLength,
		)
		return func() {
			shardedPublisherTask(testConfig)
		}
	}

	log.Info(
		"starting sharded subscriber task",
		"env", testConfig.Env,
		"num-channels", testConfig.NumChannels,
		"subs-per-channel", testConfig.NumSubscriptions,
	)
	return func() {
		shardedSubscriberTask(testConfig)
	}
}
