package main

import (
	"fmt"
	"github.com/ably/ably-go/ably"
	"github.com/ably/ably-go/ably/proto"
	"github.com/myzhan/boomer"
	"os"
	"time"
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

func waitForMessage(env string, apiKey string, channelName string) (error, *proto.Message) {
	options := ably.NewClientOptions(apiKey)
	options.Environment = env

	client, err := ably.NewRealtimeClient(options)
	if err != nil {
		return err, nil
	}
	defer client.Close()

	channel := client.Channels.Get(channelName)

	sub, err := channel.Subscribe()
	if err != nil {
		return err, nil
	}

	fmt.Println("Waiting for message...")

	msg := <-sub.MessageChannel()

	fmt.Println("Message received!")

	return nil, msg
}

func subscribeTask(env string, apiKey string, channelName string) {
	start := time.Now()

	err, msg := waitForMessage(env, apiKey, channelName)
	_ = msg

	elapsed := time.Since(start)

	if err == nil {
		boomer.RecordSuccess("ably", "subscribe", elapsed.Nanoseconds()/int64(time.Millisecond), int64(10))
	} else {
		boomer.RecordFailure("ably", "subscribe", elapsed.Nanoseconds()/int64(time.Millisecond), err.Error())
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
