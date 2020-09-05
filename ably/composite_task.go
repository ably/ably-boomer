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

	var wg sync.WaitGroup
	errorChannel := make(chan error)

	client, err := newAblyClient(testConfig)
	if err != nil {
		log.Println("Subscribe Error - " + err.Error())
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer client.Close()

	compositeUserMutex.Lock()
	compositeUserCounter++
	userNumber := compositeUserCounter
	compositeUserMutex.Unlock()

	shardedChannelName := generateShardedChannelName(testConfig, userNumber)

	shardedChannel := client.Channels.Get(shardedChannelName)
	defer shardedChannel.Close()

	shardedSub, err := shardedChannel.Subscribe()
	if err != nil {
		log.Println("Subscribe Error - " + err.Error())
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}

	wg.Add(1)
	go reportSubscriptionToLocust(ctx, shardedSub, client.Connection, errorChannel, &wg)

	personalChannelName := randomString(100)
	personalChannel := client.Channels.Get(personalChannelName)
	defer personalChannel.Close()

	for i := 0; i < testConfig.NumSubscriptions; i++ {
		personalSub, err := personalChannel.Subscribe()
		if err != nil {
			log.Println("Subscribe Error - " + err.Error())
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}

		wg.Add(1)
		go reportSubscriptionToLocust(ctx, personalSub, client.Connection, errorChannel, &wg)
	}

	wg.Add(1)
	go publishOnInterval(ctx, testConfig, personalChannel, rand.Intn(testConfig.PublishInterval), errorChannel, &wg)

	select {
	case err := <-errorChannel:
		log.Println(err)
		cancel()
		wg.Wait()
		client.Close()
		return
	case <-ctx.Done():
		wg.Wait()
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
