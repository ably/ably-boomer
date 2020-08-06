package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/ably/perf"
	"github.com/ably/ably-go/ably"
)

func generateChannelName(testConfig TestConfig, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%testConfig.NumChannels)
}

func publishOnInterval(ctx context.Context, l perf.LocustReporter, testConfig TestConfig, channel *ably.RealtimeChannel, delay int) {
	log.Println("Delaying publish to", channel.Name, "for", delay+testConfig.PublishInterval, "seconds")
	time.Sleep(time.Duration(delay) * time.Second)

	ticker := time.NewTicker(time.Duration(testConfig.PublishInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Publishing to:", channel.Name)

			data := randomString(testConfig.MessageDataLength)
			timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

			_, err := channel.Publish(timePublished, data)

			if err != nil {
				l.RecordFailure("ably", "publish", 0, err.Error())
			} else {
				l.RecordSuccess("ably", "publish", 0, 0)
			}
		case <-ctx.Done():
			return
		}
	}
}

func shardedPublisherTask(testConfig TestConfig, l perf.LocustReporter) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	client, err := newAblyClient(testConfig)
	if err != nil {
		l.RecordFailure("ably", "publish", 0, err.Error())
		return
	}
	defer client.Close()

	for i := 0; i < testConfig.NumChannels; i++ {
		channelName := generateChannelName(testConfig, i)

		channel := client.Channels.Get(channelName)
		defer channel.Close()

		delay := i % testConfig.PublishInterval

		go publishOnInterval(ctx, l, testConfig, channel, delay)
	}

	<-ctx.Done()

	client.Close()
}

func shardedSubscriberTask(testConfig TestConfig, l perf.LocustReporter) {
	ctx, cancel := context.WithCancel(context.Background())
	boomer.Events.Subscribe("boomer:stop", cancel)

	errorChannel := make(chan error)

	clients := []ably.RealtimeClient{}

	for i := 0; i < testConfig.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			client, err := newAblyClient(testConfig)
			if err != nil {
				l.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer client.Close()

			clients = append(clients, *client)

			channelName := generateChannelName(testConfig, i)

			log.Println("Subscribing to channel:", channelName)

			channel := client.Channels.Get(channelName)
			defer channel.Close()

			sub, err := channel.Subscribe()
			if err != nil {
				l.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer sub.Close()

			go reportSubscriptionToLocust(ctx, l, sub, client.Connection, errorChannel)
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
			log.Println(err)
			cleanup()
			return
		case <-ctx.Done():
			cleanup()
			return
		}
	}
}

func curryShardedTask(testConfig TestConfig, l perf.LocustReporter) func() {
	log.Println("Test Type: Sharded")
	log.Println("Ably Env:", testConfig.Env)
	log.Println("Number of Channels:", testConfig.NumChannels)
	log.Println("Number of Subscriptions:", testConfig.NumSubscriptions)
	log.Println("Publisher:", testConfig.Publisher)

	if testConfig.Publisher {
		log.Println("Publish Interval:", testConfig.PublishInterval, "seconds")

		return func() {
			shardedPublisherTask(testConfig, l)
		}
	}

	return func() {
		shardedSubscriberTask(testConfig, l)
	}
}
