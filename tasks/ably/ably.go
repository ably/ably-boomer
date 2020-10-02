package ably

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
	ablyrpc "github.com/ably/ably-go/ably/proto"
	"github.com/inconshreveable/log15"
)

// Conf is the task's configuration.
type Conf struct {
	Logger           log15.Logger
	APIKey           string
	Env              string
	ChannelName      string
	NumChannels      int
	MsgDataLength    int
	NumSubscriptions int
	PublishInterval  int
}

// Task contains all data required to run an Ably Runtime task.
type Task struct {
	conf                   Conf
	letters                []rune
	userCounter            int
	userMutex              sync.Mutex
	errorMsgTimestampRegex *regexp.Regexp
}

// NewTask returns a new Ably Runtime task.
func NewTask(conf Conf) *Task {
	return &Task{
		conf:                   conf,
		letters:                []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		errorMsgTimestampRegex: regexp.MustCompile(`tamp=[0-9]+`),
	}
}

// Run starts the task.
func (t *Task) Run() {
	log := t.conf.Logger
	log.Info(
		"starting task",
		"env", t.conf.Env,
		"num-channels", t.conf.NumChannels,
		"subs-per-channel", t.conf.NumSubscriptions,
		"publish-interval", t.conf.PublishInterval,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	var wg sync.WaitGroup
	errorChannel := make(chan error)

	log.Info("creating realtime connection")
	client, err := newAblyClient(t.conf.APIKey, t.conf.Env)
	if err != nil {
		log.Error("error creating realtime connection", "err", err)

		errMsg := t.errorMsgTimestampRegex.ReplaceAllString(err.Error(), "tamp=<timestamp>")

		boomer.RecordFailure("ably", "subscribe", 0, errMsg)
		return
	}
	defer client.Close()

	t.userMutex.Lock()
	t.userCounter++
	userNumber := t.userCounter
	t.userMutex.Unlock()

	shardedChannelName := generateChannelName(t.conf.NumChannels, userNumber)

	shardedChannel := client.Channels.Get(shardedChannelName)
	defer shardedChannel.Close()

	log.Info("creating sharded channel subscriber", "name", shardedChannelName)
	shardedSub, err := shardedChannel.Subscribe()
	if err != nil {
		log.Error("error creating sharded channel subscriber", "name", shardedChannelName, "err", err)

		errMsg := t.errorMsgTimestampRegex.ReplaceAllString(err.Error(), "tamp=<timestamp>")

		boomer.RecordFailure("ably", "subscribe", 0, errMsg)
		return
	}

	wg.Add(1)
	go t.reportSubscriptionToLocust(ctx, shardedSub, client.Connection, errorChannel, &wg, log.New("channel", shardedChannelName))

	personalChannelName := t.randomString(100)
	personalChannel := client.Channels.Get(personalChannelName)
	defer personalChannel.Close()

	log.Info("creating personal subscribers", "channel", personalChannelName, "count", t.conf.NumSubscriptions)
	for i := 0; i < t.conf.NumSubscriptions; i++ {
		log.Info("creating personal subscriber", "num", i+1, "name", personalChannelName)
		personalSub, err := personalChannel.Subscribe()
		if err != nil {
			log.Error("error creating personal subscriber", "num", i+1, "name", personalChannelName, "err", err)
			boomer.RecordFailure("ably", "subscribe", 0, err.Error())
			return
		}

		wg.Add(1)
		go t.reportSubscriptionToLocust(ctx, personalSub, client.Connection, errorChannel, &wg, log.New("channel", personalChannelName))
	}

	log.Info("creating publishers", "count", t.conf.NumChannels)
	for i := 0; i < t.conf.NumChannels; i++ {
		channelName := generateChannelName(t.conf.NumChannels, i)

		channel := client.Channels.Get(channelName)
		defer channel.Close()

		delay := i % t.conf.PublishInterval

		log.Info("starting publisher", "num", i+1, "channel", channelName, "delay", delay)
		wg.Add(1)
		go t.publishOnInterval(ctx, t.conf.PublishInterval, t.conf.MsgDataLength, channel, delay, errorChannel, &wg, log)
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

func generateChannelName(numChannels, number int) string {
	return "test-channel-" + strconv.Itoa(number%numChannels)
}

func newAblyClient(apiKey, env string) (*ably.RealtimeClient, error) {
	options := ably.NewClientOptions(apiKey)
	options.Environment = env

	return ably.NewRealtimeClient(options)
}

func millisecondTimestamp() int64 {
	nanos := time.Now().UnixNano()
	millis := nanos / int64(time.Millisecond)
	return millis
}

func (t *Task) randomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = t.letters[rand.Intn(len(t.letters))]
	}
	return string(b)
}

func randomDelay(log log15.Logger) {
	r := rand.Intn(60)
	log.Info("introducing random delay", "seconds", r)
	time.Sleep(time.Duration(r) * time.Second)
	log.Info("continuing after random delay")
}

func (t *Task) publishOnInterval(
	ctx context.Context,
	publishInterval,
	msgDataLength int,
	channel *ably.RealtimeChannel,
	delay int,
	errorChannel chan<- error,
	wg *sync.WaitGroup,
	log log15.Logger,
) {
	log = log.New("channel", channel.Name)
	log.Info("creating publisher", "period", publishInterval)

	log.Info("introducing random delay before starting to publish", "seconds", delay)
	time.Sleep(time.Duration(delay) * time.Second)
	log.Info("continuing after random delay")

	ticker := time.NewTicker(time.Duration(publishInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data := t.randomString(msgDataLength)
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
			}

			boomer.RecordSuccess("ably", "publish", 0, 0)
		case <-ctx.Done():
			ticker.Stop()
			wg.Done()
			return
		}
	}
}

func (t *Task) reportSubscriptionToLocust(
	ctx context.Context,
	sub *ably.Subscription,
	conn *ably.Conn,
	errorChannel chan<- error,
	wg *sync.WaitGroup,
	log log15.Logger,
) {
	connectionStateChannel := make(chan ably.State)
	conn.On(connectionStateChannel)

	var lastDisconnectTime int64 = 0

	for {
		select {
		case connState, ok := <-connectionStateChannel:
			if !ok {
				log.Warn("connection state channel closed", "id", conn.ID())
				errorChannel <- errors.New("connection state channel closed")
				continue
			}

			log.Info(
				"connection state changed",
				"id", conn.ID(),
				"key", conn.Key(),
				"state", connState.State,
				"err", connState.Err,
			)

			if connState.State == ably.StateConnDisconnected {
				lastDisconnectTime = millisecondTimestamp()
			} else if connState.State == ably.StateConnConnected && lastDisconnectTime != 0 {
				timeDisconnected := millisecondTimestamp() - lastDisconnectTime

				log.Info("reporting reconnect time", "id", conn.ID(), "duration", timeDisconnected)
				boomer.RecordSuccess("ably", "reconnect", timeDisconnected, 0)
			}
		case <-ctx.Done():
			log.Info("subscriber context done", "id", conn.ID())
			wg.Done()
			return
		case msg, ok := <-sub.MessageChannel():
			if !ok {
				log.Warn("subscriber message channel closed", "id", conn.ID())
				errorChannel <- errors.New("subscriber message channel closed")
				continue
			}
			validateMsg(msg, log)
		}
	}
}

func validateMsg(msg *ablyrpc.Message, log log15.Logger) {
	timePublished, err := strconv.ParseInt(msg.Name, 10, 64)
	if err != nil {
		log.Error("error parsing message name as timestamp", "err", err)
		boomer.RecordFailure("ably", "subscribe", 0, err.Error())
		return
	}

	timeElapsed := millisecondTimestamp() - timePublished
	bytes := len(fmt.Sprint(msg.Data))

	log.Info("received message",  "size", bytes, "latency", timeElapsed)
	boomer.RecordSuccess("ably", "subscribe", timeElapsed, int64(bytes))
}
