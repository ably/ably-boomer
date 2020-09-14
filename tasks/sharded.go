package tasks

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
)

type ShardedConf struct {
	APIKey           string
	Env              string
	NumChannels      int
	PublishInterval  int
	MsgDataLength    int
	NumSubscriptions int
	Publisher        bool
}

func generateChannelName(numChannels, number int) string {
	return "sharded-test-channel-" + strconv.Itoa(number%numChannels)
}

func publishOnInterval(ctx context.Context, publishInterval, msgDataLength int, channel *ably.RealtimeChannel, delay int) {
	log.Println("Delaying publish to", channel.Name, "for", delay+publishInterval, "seconds")
	time.Sleep(time.Duration(delay) * time.Second)
	log.Info("continuing after random delay")

	ticker := time.NewTicker(time.Duration(publishInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Publishing to:", channel.Name)

			data := randomString(msgDataLength)
			timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

			log.Info("publishing message", "size", len(data))
			_, err := channel.Publish(timePublished, data)
			if err != nil {
				log.Error("error publishing message", "err", err)
				boomer.RecordFailure("ably", "publish", 0, err.Error())
				errorChannel <- err
				ticker.Stop()
				wg.Done()
				return
			} else {
				boomer.RecordSuccess("ably", "publish", 0, 0)
			}
		case <-ctx.Done():
			ticker.Stop()
			wg.Done()
			return
		}
	}
}

func shardedPublisherTask(conf ShardedConf) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errorChannel := make(chan error)
	var wg sync.WaitGroup

	boomer.Events.Subscribe("boomer:stop", cancel)

	client, err := newAblyClient(conf.APIKey, conf.Env)
	if err != nil {
		log.Error("error creating realtime connection", "err", err)
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		return
	}
	defer client.Close()

	for i := 0; i < conf.NumChannels; i++ {
		channelName := generateChannelName(conf.NumChannels, i)

		channel := client.Channels.Get(channelName)
		defer channel.Close()

		delay := i % conf.PublishInterval

		go publishOnInterval(ctx, conf.PublishInterval, conf.MsgDataLength, channel, delay)
	}

	select {
	case err := <-errorChannel:
		log.Error("error from publisher goroutine", "err", err)
		cancel()
		client.Close()
		shardedPublisherTask(testConfig)
	case <-ctx.Done():
		log.Info("sharded publisher task context done, cleaning up")
		cancel()
		client.Close()
		return
	}
}

func shardedSubscriberTask(conf ShardedConf) {
	ctx, cancel := context.WithCancel(context.Background())
	boomer.Events.Subscribe("boomer:stop", cancel)

	errorChannel := make(chan error)
	var wg sync.WaitGroup

	clients := []ably.RealtimeClient{}

	for i := 0; i < conf.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			client, err := newAblyClient(conf.APIKey, conf.Env)
			if err != nil {
				log.Error("error creating realtime connection", "num", i+1, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer client.Close()

			clients = append(clients, *client)

			channelName := generateChannelName(conf.NumChannels, i)

			channel := client.Channels.Get(channelName)
			defer channel.Close()

			log.Info("creating subscriber", "num", i+1, "name", channelName)
			sub, err := channel.Subscribe()
			if err != nil {
				log.Error("error creating subscriber", "num", i+1, "name", channelName, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer sub.Close()

			go reportSubscriptionToLocust(ctx, sub, client.Connection, errorChannel, &wg, log.New("channel", channelName))
		}
	}

	cleanup := func() {
		for _, client := range clients {
			client.Close()
		}
	}

	for {
		select {
		case err := <-errorChannel:
			log.Error("error from subscriber goroutine", "err", err)
			cleanup()
			return
		case <-ctx.Done():
			log.Info("sharded subscriber task context done, cleaning up")
			cleanup()
			return
		}
	}
}

func CurryShardedTask(conf ShardedConf) func() {
	log.Println("Test Type: Sharded")
	log.Println("Ably Env:", conf.Env)
	log.Println("Number of Channels:", conf.NumChannels)
	log.Println("Number of Subscriptions:", conf.NumSubscriptions)
	log.Println("Publisher:", conf.Publisher)

	if conf.Publisher {
		log.Println("Publish Interval:", conf.PublishInterval, "seconds")

		return func() {
			shardedPublisherTask(conf)
		}
	}

	log.Info(
		"starting sharded subscriber task",
		"env", testConfig.Env,
		"num-channels", testConfig.NumChannels,
		"subs-per-channel", testConfig.NumSubscriptions,
	)
	return func() {
		shardedSubscriberTask(conf)
	}
}
