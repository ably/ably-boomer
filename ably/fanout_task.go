package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ably-forks/boomer"
)

func fanOutTask(testConfig TestConfig) {
	client := newAblyClient(testConfig)
	defer client.Close()

	channel := client.Channels.Get(testConfig.ChannelName)

	sub, err := channel.Subscribe()
	if err != nil {
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())

	boomer.Events.Subscribe("boomer:stop", cancel)

	for {
		select {
		case msg := <-sub.MessageChannel():
			timeElapsed := millisecondTimestamp() - msg.Timestamp
			bytes := len(fmt.Sprint(msg.Data))

			boomer.RecordSuccess("ably", "subscribe", timeElapsed, int64(bytes))
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
