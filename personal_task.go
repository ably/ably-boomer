package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
	"github.com/ably/ably-go/ably/proto"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
const channelNameLength = 50

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

func personalTask(env string, apiKey string, publishInterval int, numSubscriptions int) {
	options := ably.NewClientOptions(apiKey)
	options.Environment = env

	client, err := ably.NewRealtimeClient(options)
	if err != nil {
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer client.Close()

	channel := client.Channels.Get(randomString(channelNameLength))

  aggregateMessageChannel := make(chan *proto.Message)

  for i := 0; i < numSubscriptions; i++ {
    sub, err := channel.Subscribe()
    if err != nil {
      boomer.RecordFailure("ably", "subscribe", 0, err.Error())
      return
    }

    go func() {
      for msg := range sub.MessageChannel() {
        aggregateMessageChannel <- msg
      }
    }()

    defer sub.Close()
  }

	ctx, cancel := context.WithCancel(context.Background())

	boomer.Events.Subscribe("boomer:stop", cancel)

	ticker := time.NewTicker(time.Duration(publishInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
      data := randomString(50)
			res, err := channel.Publish("test", data)
			_ = res

			if err != nil {
				boomer.RecordFailure("ably", "publish", 0, err.Error())
			} else {
				boomer.RecordSuccess("ably", "publish", 0, 0)
			}
		case msg := <-aggregateMessageChannel:
			timeElapsed := millisecondTimestamp() - msg.Timestamp
			bytes := len(fmt.Sprint(msg.Data))

			boomer.RecordSuccess("ably", "subscribe", timeElapsed, int64(bytes))
		case <-ctx.Done():
			return
		}
	}
}

func curryPersonalTask() func() {
	env := ablyEnv()
	apiKey := ablyApiKey()
  publishInterval := ablyPublishInterval()
  numSubscriptions := ablyNumSubscriptions()

  log.Println("Test Type: Personal")
	log.Println("Ably Env:", env)
	log.Println("Publish Interval:", publishInterval, "seconds")
	log.Println("Subscriptions Per Channel:", numSubscriptions)

	return func() {
		personalTask(env, apiKey, publishInterval, numSubscriptions)
	}
}
