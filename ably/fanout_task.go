package ably

import (
	"context"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-go/ably"
	"github.com/inconshreveable/log15"
)

func fanOutTask(config *config.Config, log log15.Logger) {
	log.Info("creating realtime connection")
	client, err := newAblyClient(config, log)
	if err != nil {
		log.Error("error creating realtime connection", "err", err)
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	channel := client.Channels.Get(config.ChannelName)

	log.Info("creating subscriber", "name", config.ChannelName)
	msgC := make(chan *ably.Message)
	unsub, err := channel.SubscribeAll(ctx, func(msg *ably.Message) {
		select {
		case msgC <- msg:
		case <-ctx.Done():
		}
	})
	if err != nil {
		log.Error("error creating subscriber", "name", config.ChannelName, "err", err)
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer unsub()

	var wg sync.WaitGroup
	errorChannel := make(chan error)

	wg.Add(1)
	go reportSubscriptionToLocust(ctx, msgC, client.Connection, errorChannel, &wg, log.New("channel", config.ChannelName))

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

func curryFanOutTask(config *config.Config, log log15.Logger) func() {
	log.Info(
		"starting fanout task",
		"env", config.Env,
		"channel", config.ChannelName,
	)

	return func() {
		fanOutTask(config, log)
	}
}

func millisecondTimestamp() int64 {
	nanos := time.Now().UnixNano()
	millis := nanos / int64(time.Millisecond)
	return millis
}
