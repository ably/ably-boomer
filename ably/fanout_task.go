package main

import (
	"context"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
)

func fanOutTask(testConfig TestConfig) {
	log.Info("creating realtime connection")
	client, err := newAblyClient(testConfig)
	if err != nil {
		log.Error("error creating realtime connection", "err", err)
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer client.Close()

	channel := client.Channels.Get(testConfig.ChannelName)
	defer channel.Close()

	log.Info("creating subscriber", "name", testConfig.ChannelName)
	sub, err := channel.Subscribe()
	if err != nil {
		log.Error("error creating subscriber", "name", testConfig.ChannelName, "err", err)
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	var wg sync.WaitGroup
	errorChannel := make(chan error)

	wg.Add(1)
	go reportSubscriptionToLocust(ctx, sub, client.Connection, errorChannel, &wg)

	select {
	case err := <-errorChannel:
		log.Error("error from subscriber goroutine", "err", err)
		cancel()
		wg.Wait()
		client.Close()
		return
	case <-ctx.Done():
		log.Info("fanout task context done, cleaning up")
		wg.Wait()
		client.Close()
		return
	}
}

func curryFanOutTask(testConfig TestConfig) func() {
	log.Info(
		"starting fanout task",
		"env", testConfig.Env,
		"channel", testConfig.ChannelName,
	)

	return func() {
		fanOutTask(testConfig)
	}
}

func millisecondTimestamp() int64 {
	nanos := time.Now().UnixNano()
	millis := nanos / int64(time.Millisecond)
	return millis
}
