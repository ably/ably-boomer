package tasks

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
)

type PersonalConf struct {
	APIKey           string
	Env              string
	PublishInterval  int
	NumSubscriptions int
	MsgDataLength    int
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

const channelNameLength = 100

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

func randomDelay() {
	r := rand.Intn(60)
	log.Info("introducing random delay", "seconds", r)
	time.Sleep(time.Duration(r) * time.Second)
	log.Info("continuing after random delay")
}

func personalTask(conf PersonalConf) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	var wg sync.WaitGroup
	errorChannel := make(chan error)

	channelName := randomString(channelNameLength)

	subClients := []ably.RealtimeClient{}

	for i := 0; i < conf.NumSubscriptions; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			subClient, err := newAblyClient(conf.APIKey, conf.Env)
			if err != nil {
				log.Error("error creating subscriber realtime connection", "num", i+1, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer subClient.Close()

			subClients = append(subClients, *subClient)

			channel := subClient.Channels.Get(channelName)
			defer channel.Close()

			log.Info("creating subscriber", "num", i+1, "channel", channelName)
			sub, err := channel.Subscribe()
			if err != nil {
				log.Error("error creating subscriber", "num", i+1, "channel", channelName, "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				return
			}
			defer sub.Close()

			wg.Add(1)
			go reportSubscriptionToLocust(ctx, sub, subClient.Connection, errorChannel, &wg, log.New("channel", channelName))
		}
	}

	publishClient, err := newAblyClient(conf.APIKey, conf.Env)
	if err != nil {
		log.Error("error creating publisher realtime connection", "err", err)
		boomer.RecordFailure("ably", "publish", 0, err.Error())
		return
	}
	defer publishClient.Close()

	channel := publishClient.Channels.Get(channelName)
	defer channel.Close()

	cleanup := func() {
		publishClient.Close()

		for _, subClient := range subClients {
			subClient.Close()
		}
	}

	randomDelay()

	ticker := time.NewTicker(time.Duration(conf.PublishInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-errorChannel:
			log.Error("error from subscriber goroutine", "err", err)
			cancel()
			wg.Wait()
			cleanup()
			return
		case <-ticker.C:
			data := randomString(conf.MsgDataLength)
			timePublished := strconv.FormatInt(millisecondTimestamp(), 10)

			log.Info("publishing message", "size", len(data))
			_, err := channel.Publish(timePublished, data)

			if err != nil {
				log.Error("error publishing message", "err", err)
				boomer.RecordFailure("ably", "publish", 0, err.Error())
			} else {
				boomer.RecordSuccess("ably", "publish", 0, 0)
			}
		case <-ctx.Done():
			log.Info("personal task context done, cleaning up")
			wg.Wait()
			cleanup()
			return
		}
	}
}

func CurryPersonalTask(conf PersonalConf) func() {
	log.Println("Test Type: Personal")
	log.Println("Ably Env:", conf.Env)
	log.Println("Publish Interval:", conf.PublishInterval, "seconds")
	log.Println("Subscriptions Per Channel:", conf.NumSubscriptions)
	log.Println("Message Data Length:", conf.MsgDataLength, "characters")

	return func() {
		personalTask(conf)
	}
}
