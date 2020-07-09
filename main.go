package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
)

func getEnv(name string) string {
	value, exists := os.LookupEnv(name)

	if !exists {
		log.Fatalln("Environment Variable '" + name + "' not set!")
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

func millisecondTimestamp() int64 {
	nanos := time.Now().UnixNano()
	millis := nanos / int64(time.Millisecond)
	return millis
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

func currySubscribeTask() func() {
	env := ablyEnv()
	apiKey := ablyApiKey()
	channelName := ablyChannelName()

	log.Println("Ably Env:", env)
	log.Println("Channel Name:", channelName)

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
