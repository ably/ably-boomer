package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
)

func generateChannelName(testConfig TestConfig, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%testConfig.NumChannels)
}

func publishOnInterval(testConfig TestConfig, ctx context.Context, channel *ably.RealtimeChannel, delay int) {
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
				boomer.RecordFailure("ably", "publish", 0, err.Error())
			} else {
				boomer.RecordSuccess("ably", "publish", 0, 0)
			}
		case <-ctx.Done():
			return
		}
	}
}

func shardedPublisherTask(testConfig TestConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	boomer.Events.Subscribe("boomer:stop", cancel)

	for i := 0; i < testConfig.NumChannels; i++ {
		channelName := generateChannelName(testConfig, i)

		client, err := newAblyClient(testConfig)
		if err != nil {
			boomer.RecordFailure("ably", "publish", 0, err.Error())
			return
		}
		defer client.Close()

		channel := client.Channels.Get(channelName)
		defer channel.Close()

		delay := i % testConfig.PublishInterval

		go publishOnInterval(testConfig, ctx, channel, delay)
	}

	select {
	case <-ctx.Done():
		return
	}
}

func shardedSubscriberTask(testConfig TestConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	boomer.Events.Subscribe("boomer:stop", cancel)

	for i := 0; i < testConfig.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			client, err := newAblyClient(testConfig)
			if err != nil {
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer client.Close()

			channelName := generateChannelName(testConfig, i)

			log.Println("Subscribing to channel:", channelName)

			channel := client.Channels.Get(channelName)
			defer channel.Close()

			sub, err := channel.Subscribe()
			if err != nil {
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}

			go reportSubscriptionToLocust(ctx, sub, client.Connection)
		}
	}

	select {
	case <-ctx.Done():
		return
	}
}

func curryShardedTask(testConfig TestConfig) func() {
	log.Println("Test Type: Sharded")
	log.Println("Ably Env:", testConfig.Env)
	log.Println("Number of Channels:", testConfig.NumChannels)
	log.Println("Number of Subscriptions:", testConfig.NumSubscriptions)
	log.Println("Publisher:", testConfig.Publisher)

	if testConfig.Publisher {
		log.Println("Publish Interval:", testConfig.PublishInterval, "seconds")

		return func() {
			shardedPublisherTask(testConfig)
		}
	} else {
		return func() {
			shardedSubscriberTask(testConfig)
		}
	}
}
