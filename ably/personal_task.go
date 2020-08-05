package main

import (
	"context"
	"log"
	"math/rand"
	"strconv"
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

func personalTask(testConfig TestConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	errorChannel := make(chan error)

	channelName := randomString(channelNameLength)

	subClients := []ably.RealtimeClient{}

	for i := 0; i < testConfig.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			subClient, err := newAblyClient(testConfig)
			if err != nil {
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer subClient.Close()

			subClients = append(subClients, *subClient)

			channel := subClient.Channels.Get(channelName)
			defer channel.Close()

			sub, err := channel.Subscribe()
			if err != nil {
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer sub.Close()

			go reportSubscriptionToLocust(ctx, sub, subClient.Connection, errorChannel)
		}
	}

	publishClient, err := newAblyClient(testConfig)
	if err != nil {
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		return
	}
	defer publishClient.Close()

	channel := publishClient.Channels.Get(channelName)
	defer channel.Close()

	cleanup := func() {
		publishClient.Close()

		for _, subClient := range subClients {
			subClient.Close()
		}
	}

	randomDelay()

	ticker := time.NewTicker(time.Duration(testConfig.PublishInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-errorChannel:
			log.Println(err)
			cleanup()
			return
		case <-ticker.C:
			data := randomString(testConfig.MessageDataLength)
			timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

			_, err := channel.Publish(timePublished, data)

			if err != nil {
				boomer.RecordFailure("ably", "publish", 0, err.Error())
			} else {
				boomer.RecordSuccess("ably", "publish", 0, 0)
			}
		case <-ctx.Done():
			cleanup()
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
