package ably

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

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
