package main

import (
	"context"
	"log"
	"time"

	"github.com/ably-forks/boomer"
)

func fanOutTask(testConfig TestConfig) {
	client, err := newAblyClient(testConfig)

	if err != nil {
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer client.Close()

	channel := client.Channels.Get(testConfig.ChannelName)
	defer channel.Close()

	sub, err := channel.Subscribe()
	if err != nil {
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	errorChannel := make(chan error)
	go reportSubscriptionToLocust(ctx, sub, client.Connection, errorChannel)

	for {
		select {
		case err := <-errorChannel:
			log.Println(err)
			return
		case <-ctx.Done():
			return
		}
	}
}

func curryFanOutTask(testConfig TestConfig) func() {
	log.Println("Test Type: FanOut")
	log.Println("Ably Env:", testConfig.Env)
	log.Println("Channel Name:", testConfig.ChannelName)

	return func() {
		fanOutTask(testConfig)
	}
}

func millisecondTimestamp() int64 {
	nanos := time.Now().UnixNano()
	millis := nanos / int64(time.Millisecond)
	return millis
}
