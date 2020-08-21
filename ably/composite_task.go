package main

import (
	"context"
	"log"
	"strconv"
	"time"
	"math/rand"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
)

func generateShardedChannelName(testConfig TestConfig, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%testConfig.NumChannels)
}

func compositeTask(testConfig TestConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	errorChannel := make(chan error)

	clients := []ably.RealtimeClient{}

	for i := 0; i < testConfig.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(100 * time.Millisecond)

			client, err := newAblyClient(testConfig)
			defer client.Close()

			if err != nil {
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}

			clients = append(clients, *client)


			shardedChannelName := generateShardedChannelName(testConfig, i)

			shardedChannel := client.Channels.Get(shardedChannelName)
			defer shardedChannel.Close()

			log.Println("Subscribing to channel:", shardedChannelName)

			shardedSub, err := shardedChannel.Subscribe()
			if err != nil {
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}

			go reportSubscriptionToLocust(ctx, shardedSub, client.Connection, errorChannel)


			personalChannelName := randomString(100)
			personalChannel := client.Channels.Get(personalChannelName)
			defer personalChannel.Close()

			log.Println("Subscribing to channel:", personalChannelName)

			personalSub, err := personalChannel.Subscribe()
			if err != nil {
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}

			go reportSubscriptionToLocust(ctx, personalSub, client.Connection, errorChannel)

			go publishOnInterval(ctx, testConfig, personalChannel, rand.Intn(testConfig.PublishInterval), errorChannel)
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
			cancel()
			cleanup()
			return
		case <-ctx.Done():
			cancel()
			cleanup()
			return
		}
	}
}

func curryCompositeTask(testConfig TestConfig) func() {
	log.Println("Test Type: Composite")
	log.Println("Ably Env:", testConfig.Env)
	log.Println("Number of Channels:", testConfig.NumChannels)
	log.Println("Number of Subscriptions:", testConfig.NumSubscriptions)
	log.Println("Publish Interval:", testConfig.PublishInterval, "seconds")
	log.Println("Publisher:", testConfig.Publisher)

	return func() {
		compositeTask(testConfig)
	}
}
