package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
)

func retryPublish(attempts int, sleep time.Duration, channel *ably.RealtimeChannel, data string) error {
	isAttached := channel.State() == ably.StateChanAttached

	var err error

	if isAttached {
		timePublished := strconv.FormatInt(millisecondTimestamp(), 10)
		_, err = channel.Publish(timePublished, data)
	}

	if err != nil || !isAttached {
		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			return retry(attempts, sleep, channel, data)
		}
		return err
	}
	return nil
}

func generateChannelName(testConfig TestConfig, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%testConfig.NumChannels)
}

func publishOnInterval(ctx context.Context, testConfig TestConfig, channel *ably.RealtimeChannel, delay int, errorChannel chan<- error) {
	log.Println("Delaying publish to", channel.Name, "for", delay+testConfig.PublishInterval, "seconds")
	time.Sleep(time.Duration(delay) * time.Second)

	publishRetries := testConfig.PublishInterval / 2

	ticker := time.NewTicker(time.Duration(testConfig.PublishInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Publishing to:", channel.Name)

			data := randomString(testConfig.MessageDataLength)

			err := retryPublish(publishRetries, time.Second, channel, data)

			if err != nil {
				boomer.RecordFailure("ably", "publish", 0, err.Error())
				errorChannel <- err
				ticker.Stop()
				return
			} else {
				boomer.RecordSuccess("ably", "publish", 0, 0)
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func shardedPublisherTask(testConfig TestConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errorChannel := make(chan error)

	boomer.Events.Subscribe("boomer:stop", cancel)

	client, err := newAblyClient(testConfig)
	if err != nil {
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		return
	}
	defer client.Close()

	for i := 0; i < testConfig.NumChannels; i++ {
		channelName := generateChannelName(testConfig, i)

		channel := client.Channels.Get(channelName)
		defer channel.Close()

		delay := i % testConfig.PublishInterval

		go publishOnInterval(ctx, testConfig, channel, delay, errorChannel)
	}

	<-ctx.Done()

	client.Close()
}

func shardedSubscriberTask(testConfig TestConfig) {
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
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
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
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer sub.Close()

			go reportSubscriptionToLocust(ctx, sub, client.Connection, errorChannel)
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
	}

	return func() {
		shardedSubscriberTask(testConfig)
	}
}
