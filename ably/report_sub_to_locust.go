package ably

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
	"github.com/inconshreveable/log15"
)

func reportSubscriptionToLocust(ctx context.Context, sub *ably.Subscription, conn *ably.Connection, errorChannel chan<- error, wg *sync.WaitGroup, log log15.Logger) {
	connectionStateChannel := make(chan ably.ConnectionStateChange)
	off := conn.OnAll(func(event ably.ConnectionStateChange) {
		select {
		case connectionStateChannel <- event:
		case <-ctx.Done():
		}
	})
	defer off()

	var lastDisconnectTime int64 = 0

	for {
		select {
		case connState, ok := <-connectionStateChannel:
			if !ok {
				log.Warn("connection state channel closed", "id", conn.ID())
				err := errors.New("Connection State Channel Closed")
				errorChannel <- err
				continue
			}

			log.Info(
				"connection state changed",
				"id", conn.ID(),
				"key", conn.Key(),
				"event", connState.Event,
				"err", connState.Reason,
			)

			if connState.Event == ably.ConnectionEventDisconnected {
				lastDisconnectTime = millisecondTimestamp()
			} else if connState.Event == ably.ConnectionEventConnected && lastDisconnectTime != 0 {
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
				err := errors.New("Subscriber Message Channel Closed")
				errorChannel <- err
				continue
			}

			timePublished, err := strconv.ParseInt(msg.Name, 10, 64)
			if err != nil {
				log.Error("error parsing message name as timestamp", "id", conn.ID(), "err", err)
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				break
			}

			timeElapsed := millisecondTimestamp() - timePublished
			bytes := len(fmt.Sprint(msg.Data))

			log.Info("received message", "conn.id", conn.ID(), "size", bytes, "latency", timeElapsed)
			boomer.RecordSuccess("ably", "subscribe", timeElapsed, int64(bytes))
		}
	}
}
