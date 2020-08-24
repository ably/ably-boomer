package main

import (
	"context"
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/ably-forks/boomer"
)

func generateShardedChannelName(testConfig TestConfig, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%testConfig.NumChannels)
}

var compositeUserCounter int
var compositeUserMutex sync.Mutex

func compositeTask(testConfig TestConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	errorChannel := make(chan error)

	client, err := newAblyClient(testConfig)
	defer client.Close()

	if err != nil {
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}

	compositeUserMutex.Lock()
	compositeUserCounter++
	userNumber := compositeUserCounter
	compositeUserMutex.Unlock()

	shardedChannelName := generateShardedChannelName(testConfig, userNumber)

	shardedChannel := client.Channels.Get(shardedChannelName)
	defer shardedChannel.Close()

	shardedSub, err := shardedChannel.Subscribe()
	if err != nil {
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}

	go reportSubscriptionToLocust(ctx, shardedSub, client.Connection, errorChannel)

	personalChannelName := randomString(100)
	personalChannel := client.Channels.Get(personalChannelName)
	defer personalChannel.Close()

	for i := 0; i < testConfig.NumSubscriptions; i++ {
		personalSub, err := personalChannel.Subscribe()
		if err != nil {
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}

		go reportSubscriptionToLocust(ctx, personalSub, client.Connection, errorChannel)
	}

	go publishOnInterval(ctx, testConfig, personalChannel, rand.Intn(testConfig.PublishInterval), errorChannel)

	select {
	case err := <-errorChannel:
		log.Println(err)
		cancel()
		client.Close()
		return
	case <-ctx.Done():
		cancel()
		client.Close()
		return
	}
}

func curryCompositeTask(testConfig TestConfig) func() {
	log.Println("Test Type: Composite")
	log.Println("Ably Env:", testConfig.Env)
	log.Println("Number of Channels:", testConfig.NumChannels)
	log.Println("Publish Interval:", testConfig.PublishInterval, "seconds")

	return func() {
		compositeTask(testConfig)
	}
}
