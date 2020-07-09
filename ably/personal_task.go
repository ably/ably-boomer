package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

const channelNameLength = 100

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randomDelay() {
	r := rand.Intn(60)
	time.Sleep(time.Duration(r) * time.Second)
}

func reportSubscriptionToLocust(ctx context.Context, sub *ably.Subscription) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-sub.MessageChannel():
			timeElapsed := millisecondTimestamp() - msg.Timestamp
			bytes := len(fmt.Sprint(msg.Data))

			boomer.RecordSuccess("ably", "subscribe", timeElapsed, int64(bytes))
		}
	}
}

func personalTask(testConfig TestConfig) {
	randomDelay()

	ctx, cancel := context.WithCancel(context.Background())
	boomer.Events.Subscribe("boomer:stop", cancel)

	channelName := randomString(channelNameLength)

	for i := 0; i < testConfig.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			subClient := newAblyClient(testConfig)
			defer subClient.Close()

			channel := subClient.Channels.Get(channelName)

			sub, err := channel.Subscribe()
			if err != nil {
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer sub.Close()

			go reportSubscriptionToLocust(ctx, sub)
		}
	}

	ticker := time.NewTicker(time.Duration(testConfig.PublishInterval) * time.Second)
	defer ticker.Stop()

	publishClient := newAblyClient(testConfig)
	defer publishClient.Close()

	channel := publishClient.Channels.Get(channelName)

	for {
		select {
		case <-ticker.C:
			data := randomString(testConfig.MessageDataLength)

			_, err := channel.Publish("test-event", data)

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

func curryPersonalTask(testConfig TestConfig) func() {
	log.Println("Test Type: Personal")
	log.Println("Ably Env:", testConfig.Env)
	log.Println("Publish Interval:", testConfig.PublishInterval, "seconds")
	log.Println("Subscriptions Per Channel:", testConfig.NumSubscriptions)
	log.Println("Message Data Length:", testConfig.MessageDataLength, "characters")

	return func() {
		personalTask(testConfig)
	}
}
