package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ably/ably-boomer/ably/perf"
	"github.com/ably/ably-go/ably"
)

func reportSubscriptionToLocust(ctx context.Context, perf *perf.Reporter, sub *ably.Subscription, conn *ably.Conn, errorChannel chan<- error) {
	connectionStateChannel := make(chan ably.State)
	conn.On(connectionStateChannel)

	var lastDisconnectTime int64 = 0

	for {
		select {
		case connState, ok := <-connectionStateChannel:
			if !ok {
				err := errors.New("Connection State Channel Closed")
				errorChannel <- err
				continue
			}

			if connState.State == ably.StateConnDisconnected {
				lastDisconnectTime = millisecondTimestamp()
			} else if connState.State == ably.StateConnConnected && lastDisconnectTime != 0 {
				timeDisconnected := millisecondTimestamp() - lastDisconnectTime

				perf.RecordSuccess("ably", "reconnect", timeDisconnected, 0)
			}
		case <-ctx.Done():
			return
		case msg, ok := <-sub.MessageChannel():
			if !ok {
				err := errors.New("Sub Message Channel Closed")
				errorChannel <- err
				continue
			}

			timePublished, err := strconv.ParseInt(msg.Name, 10, 64)

			if err != nil {
				perf.RecordFailure("ably", "subscribe", 0, err.Error())
				break
			}

			timeElapsed := millisecondTimestamp() - timePublished
			bytes := len(fmt.Sprint(msg.Data))

			perf.RecordSuccess("ably", "subscribe", timeElapsed, int64(bytes))
		}
	}
}
