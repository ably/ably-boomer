package ably

import (
	"context"
	"math/rand"
	"regexp"
	"strconv"
	"sync"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-go/ably"
	"github.com/inconshreveable/log15"
)

func generateShardedChannelName(config *config.Config, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%config.NumChannels)
}

var compositeUserCounter int
var compositeUserMutex sync.Mutex

var errorMsgTimestampRegex = regexp.MustCompile(`tamp=[0-9]+`)

func compositeTask(config *config.Config, log log15.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	var wg sync.WaitGroup
	errorChannel := make(chan error)

	log.Info("creating realtime connection")
	client, err := newAblyClient(config, log)
	if err != nil {
		log.Error("error creating realtime connection", "err", err)

		errMsg := errorMsgTimestampRegex.ReplaceAllString(err.Error(), "tamp=<timestamp>")

		boomer.RecordFailure("ably", "subscribe", 0, errMsg)
		return
	}
	defer client.Close()

	compositeUserMutex.Lock()
	compositeUserCounter++
	userNumber := compositeUserCounter
	compositeUserMutex.Unlock()

	shardedChannelName := generateShardedChannelName(config, userNumber)

	shardedChannel := client.Channels.Get(shardedChannelName)

	log.Info("creating sharded channel subscriber", "name", shardedChannelName)
	msgC := make(chan *ably.Message)
	unsub, err := shardedChannel.SubscribeAll(ctx, func(msg *ably.Message) {
		select {
		case msgC <- msg:
		case <-ctx.Done():
		}
	})
	if err != nil {
		log.Error("error creating sharded channel subscriber", "name", shardedChannelName, "err", err)

		errMsg := errorMsgTimestampRegex.ReplaceAllString(err.Error(), "tamp=<timestamp>")

		boomer.RecordFailure("ably", "subscribe", 0, errMsg)
		return
	}
	defer unsub()

	wg.Add(1)
	go reportSubscriptionToLocust(ctx, msgC, client.Connection, errorChannel, &wg, log.New("channel", shardedChannelName))

	personalChannelName := randomString(100)
	personalChannel := client.Channels.Get(personalChannelName)

	log.Info("creating personal subscribers", "channel", personalChannelName, "count", config.NumSubscriptions)
	for i := 0; i < config.NumSubscriptions; i++ {
		log.Info("creating personal subscriber", "num", i+1, "name", personalChannelName)
		msgC := make(chan *ably.Message)
		unsub, err := personalChannel.SubscribeAll(ctx, func(msg *ably.Message) {
			select {
			case msgC <- msg:
			case <-ctx.Done():
			}
		})
		if err != nil {
			log.Error("error creating personal subscriber", "num", i+1, "name", personalChannelName, "err", err)
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}
		defer unsub()

		wg.Add(1)
		go reportSubscriptionToLocust(ctx, msgC, client.Connection, errorChannel, &wg, log.New("channel", personalChannelName))
	}

	log.Info("creating personal publisher", "channel", personalChannelName)
	wg.Add(1)
	go publishOnInterval(ctx, config, personalChannel, rand.Intn(config.PublishInterval), errorChannel, &wg, log)

	select {
	case err := <-errorChannel:
		log.Error("error from subscriber or publisher goroutine", "err", err)
		cancel()
		wg.Wait()
		client.Close()
		return
	case <-ctx.Done():
		log.Info("composite task context done, cleaning up")
		wg.Wait()
		client.Close()
		return
	}
}

func curryCompositeTask(config *config.Config, log log15.Logger) func() {
	log.Info(
		"starting composite task",
		"env", config.Env,
		"num-channels", config.NumChannels,
		"subs-per-channel", config.NumSubscriptions,
		"publish-interval", config.PublishInterval,
	)

	return func() {
		compositeTask(config, log)
	}
}
