package ably

import (
	"context"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/inconshreveable/log15"
)

type FanOutConf struct {
	Logger      log15.Logger
	APIKey      string
	Env         string
	ChannelName string
}

func fanOutTask(conf FanOutConf) {
	log := conf.Logger
	log.Info("creating realtime connection")
	client, err := newAblyClient(conf.APIKey, conf.Env)
	if err != nil {
		log.Error("error creating realtime connection", "err", err)
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}
	defer client.Close()

	channel := client.Channels.Get(conf.ChannelName)
	defer channel.Close()

	log.Info("creating subscriber", "name", conf.ChannelName)
	sub, err := channel.Subscribe()
	if err != nil {
		log.Error("error creating subscriber", "name", conf.ChannelName, "err", err)
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
	go reportSubscriptionToLocust(ctx, sub, client.Connection, errorChannel, &wg, log.New("channel", conf.ChannelName))

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

func CurryFanOutTask(conf FanOutConf) func() {
	log := conf.Logger
	log.Info(
		"starting fanout task",
		"env", conf.Env,
		"channel", conf.ChannelName,
	)

	return func() {
		fanOutTask(conf)
	}
}

func millisecondTimestamp() int64 {
	nanos := time.Now().UnixNano()
	millis := nanos / int64(time.Millisecond)
	return millis
}
