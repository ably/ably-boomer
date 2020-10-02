package ably

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
	ablyrpc "github.com/ably/ably-go/ably/proto"
	"github.com/inconshreveable/log15"
)

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

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randomDelay(log log15.Logger) {
	r := rand.Intn(60)
	log.Info("introducing random delay", "seconds", r)
	time.Sleep(time.Duration(r) * time.Second)
	log.Info("continuing after random delay")
}

func publishOnInterval(
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
			}

			boomer.RecordSuccess("ably", "publish", 0, 0)
		case <-ctx.Done():
			ticker.Stop()
			wg.Done()
			return
		}
	}
}

func reportSubscriptionToLocust(
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
