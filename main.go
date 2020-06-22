package main

import (
	"fmt"
	"github.com/ably/ably-go/ably"
	"github.com/myzhan/boomer"
	"os"
	"context"
)

func getEnv(name string) string {
	value, exists := os.LookupEnv(name)

	if !exists {
		panic("Environment Variable '" + name + "' not set!")
	}

	return value
}

func ablyEnv() string {
	return getEnv("ABLY_ENV")
}

func ablyApiKey() string {
	return getEnv("ABLY_API_KEY")
}

func ablyChannelName() string {
	return getEnv("ABLY_CHANNEL_NAME")
}

func subscribeTask(env string, apiKey string, channelName string) {
	options := ably.NewClientOptions(apiKey)
	options.Environment = env

	client, err := ably.NewRealtimeClient(options)
	if err != nil {
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer client.Close()

	channel := client.Channels.Get(channelName)

	sub, err := channel.Subscribe()
	if err != nil {
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())

	boomer.Events.Subscribe("boomer:stop", cancel)

	fmt.Println("Waiting for messages...")

	for {
		select {
		case msg := <-sub.MessageChannel():
			_ = msg
			boomer.RecordSuccess("ably", "subscribe", 1, int64(10))
		case <-ctx.Done():
			return
		}
	}
}

func currySubscribeTask() func() {
	env := ablyEnv()
	apiKey := ablyApiKey()
	channelName := ablyChannelName()

	fmt.Println("Ably Env:", env)
	fmt.Println("Channel Name:", channelName)

	return func() {
		subscribeTask(env, apiKey, channelName)
	}
}

func main() {
	task := &boomer.Task{
		Name: "subscribe",
		Fn:   currySubscribeTask(),
	}

	boomer.Run(task)
}
