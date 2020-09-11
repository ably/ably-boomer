package tasks

import (
	"context"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
)

func fanOutTask(apiKey, env, channelName string) {
	client, err := newAblyClient(apiKey, env)

	if err != nil {
		log.Error("error creating realtime connection", "err", err)
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer client.Close()

	channel := client.Channels.Get(channelName)
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
	go reportSubscriptionToLocust(ctx, sub, client.Connection, errorChannel, &wg, log.New("channel", testConfig.ChannelName))

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

func CurryFanOutTask(apiKey, env, channelName string) func() {
	log.Println("Test Type: FanOut")
	log.Println("Ably Env:", env)
	log.Println("Channel Name:", channelName)

	return func() {
		fanOutTask(apiKey, env, channelName)
	}
}

func millisecondTimestamp() int64 {
	nanos := time.Now().UnixNano()
	millis := nanos / int64(time.Millisecond)
	return millis
}
