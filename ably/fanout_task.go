package main

import (
	"context"
	"log"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/ably/perf"
)

func fanOutTask(testConfig TestConfig, l perf.LocustReporter) {
	client, err := newAblyClient(testConfig)

	if err != nil {
		l.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer client.Close()

	channel := client.Channels.Get(testConfig.ChannelName)
	defer channel.Close()

	sub, err := channel.Subscribe()
	if err != nil {
		l.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	errorChannel := make(chan error)
	go reportSubscriptionToLocust(ctx, l, sub, client.Connection, errorChannel)

	select {
	case err := <-errorChannel:
		log.Println(err)
		client.Close()
		return
	case <-ctx.Done():
		client.Close()
		return
	}
}

func curryFanOutTask(testConfig TestConfig, l perf.LocustReporter) func() {
	log.Println("Test Type: FanOut")
	log.Println("Ably Env:", testConfig.Env)
	log.Println("Channel Name:", testConfig.ChannelName)

	return func() {
		fanOutTask(testConfig, l)
	}
}

func millisecondTimestamp() int64 {
	nanos := time.Now().UnixNano()
	millis := nanos / int64(time.Millisecond)
	return millis
}
