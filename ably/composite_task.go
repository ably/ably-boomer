package main

import (
	"context"
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

	log.Info("creating realtime connection")
	client, err := newAblyClient(testConfig)
	if err != nil {
		log.Error("error creating realtime connection", "err", err)
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

	log.Info("creating sharded channel subscriber", "name", shardedChannelName)
	shardedSub, err := shardedChannel.Subscribe()
	if err != nil {
		log.Error("error creating sharded channel subscriber", "name", shardedChannelName, "err", err)
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}

	wg.Add(1)
	go reportSubscriptionToLocust(ctx, shardedSub, client.Connection, errorChannel, &wg)

	personalChannelName := randomString(100)
	personalChannel := client.Channels.Get(personalChannelName)
	defer personalChannel.Close()

	log.Info("creating personal subscribers", "channel", personalChannelName, "count", testConfig.NumSubscriptions)
	for i := 0; i < testConfig.NumSubscriptions; i++ {
		log.Info("creating personal subscriber", "num", i+1, "name", personalChannelName)
		personalSub, err := personalChannel.Subscribe()
		if err != nil {
			log.Error("error creating personal subscriber", "num", i+1, "name", personalChannelName, "err", err)
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}

		wg.Add(1)
		go reportSubscriptionToLocust(ctx, personalSub, client.Connection, errorChannel, &wg)
	}

	log.Info("creating personal publisher", "channel", personalChannelName)
	wg.Add(1)
	go publishOnInterval(ctx, testConfig, personalChannel, rand.Intn(testConfig.PublishInterval), errorChannel, &wg)

	select {
	case err := <-errorChannel:
		log.Error("error from subscriber or publisher goroutine", "err", err)
		cancel()
		wg.Wait()
		client.Close()
		return
	case <-ctx.Done():
		log.Info("composite task context done, cleaning up")
		wg.Wait()
		client.Close()
		return
	}
}

func curryCompositeTask(testConfig TestConfig) func() {
	log.Info(
		"starting composite task",
		"env", testConfig.Env,
		"num-channels", testConfig.NumChannels,
		"subs-per-channel", testConfig.NumSubscriptions,
		"publish-interval", testConfig.PublishInterval,
	)

	return func() {
		compositeTask(testConfig)
	}
}
