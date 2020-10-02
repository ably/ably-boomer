package ably

import (
	"context"
	"regexp"
	"strconv"
	"sync"

	"github.com/ably-forks/boomer"
	"github.com/inconshreveable/log15"
)

// CompositeConf is the Composite task's configuration.
type CompositeConf struct {
	Logger           log15.Logger
	APIKey           string
	Env              string
	ChannelName      string
	NumChannels      int
	MsgDataLength    int
	NumSubscriptions int
	PublishInterval  int
}

func generateShardedChannelName(numChannels, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%numChannels)
}

var compositeUserCounter int
var compositeUserMutex sync.Mutex

var errorMsgTimestampRegex = regexp.MustCompile(`tamp=[0-9]+`)

func compositeTask(conf CompositeConf) {
	log := conf.Logger
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	var wg sync.WaitGroup
	errorChannel := make(chan error)

	log.Info("creating realtime connection")
	client, err := newAblyClient(conf.APIKey, conf.Env)
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

	shardedChannelName := generateShardedChannelName(conf.NumChannels, userNumber)

	shardedChannel := client.Channels.Get(shardedChannelName)
	defer shardedChannel.Close()

	log.Info("creating sharded channel subscriber", "name", shardedChannelName)
	shardedSub, err := shardedChannel.Subscribe()
	if err != nil {
		log.Error("error creating sharded channel subscriber", "name", shardedChannelName, "err", err)

		errMsg := errorMsgTimestampRegex.ReplaceAllString(err.Error(), "tamp=<timestamp>")

		boomer.RecordFailure("ably", "subscribe", 0, errMsg)
		return
	}

	wg.Add(1)
	go reportSubscriptionToLocust(ctx, shardedSub, client.Connection, errorChannel, &wg, log.New("channel", shardedChannelName))

	personalChannelName := randomString(100)
	personalChannel := client.Channels.Get(personalChannelName)
	defer personalChannel.Close()

	log.Info("creating personal subscribers", "channel", personalChannelName, "count", conf.NumSubscriptions)
	for i := 0; i < conf.NumSubscriptions; i++ {
		log.Info("creating personal subscriber", "num", i+1, "name", personalChannelName)
		personalSub, err := personalChannel.Subscribe()
		if err != nil {
			log.Error("error creating personal subscriber", "num", i+1, "name", personalChannelName, "err", err)
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}

		wg.Add(1)
		go reportSubscriptionToLocust(ctx, personalSub, client.Connection, errorChannel, &wg, log.New("channel", personalChannelName))
	}

	log.Info("creating publishers", "count", conf.NumChannels)
	for i := 0; i < conf.NumChannels; i++ {
		channelName := generateShardedChannelName(conf.NumChannels, i)

		channel := client.Channels.Get(channelName)
		defer channel.Close()

		delay := i % conf.PublishInterval

		log.Info("starting publisher", "num", i+1, "channel", channelName, "delay", delay)
		wg.Add(1)
		go publishOnInterval(ctx, conf.PublishInterval, conf.MsgDataLength, channel, delay, errorChannel, &wg, log)
	}

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

// CurryCompositeTask returns a function allowing to run the Composite task.
func CurryCompositeTask(conf CompositeConf) func() {
	log := conf.Logger
	log.Info(
		"starting composite task",
		"env", conf.Env,
		"num-channels", conf.NumChannels,
		"subs-per-channel", conf.NumSubscriptions,
		"publish-interval", conf.PublishInterval,
	)

	return func() {
		compositeTask(conf)
	}
}
