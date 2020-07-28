package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-go/ably"
)

func reportSubscriptionToLocust(ctx context.Context, sub *ably.Subscription, conn *ably.Conn) {
	defer sub.Close()

	connectionStateChannel := make(chan ably.State)
	conn.On(connectionStateChannel)

	var lastDisconnectTime int64 = 0

	for {
		select {
		case connState := <-connectionStateChannel:
			if connState.State == ably.StateConnDisconnected {
				lastDisconnectTime = millisecondTimestamp()
			} else if connState.State == ably.StateConnConnected && lastDisconnectTime != 0 {
				timeDisconnected := millisecondTimestamp() - lastDisconnectTime

				boomer.RecordSuccess("ably", "reconnect", timeDisconnected, 0)
			}
		case <-ctx.Done():
			return
		case msg := <-sub.MessageChannel():
			timePublished, err := strconv.ParseInt(msg.Name, 10, 64)

			if err != nil {
				boomer.RecordFailure("ably", "subscribe", 0, err.Error())
				break
			}

			timeElapsed := millisecondTimestamp() - timePublished
			bytes := len(fmt.Sprint(msg.Data))

			boomer.RecordSuccess("ably", "subscribe", timeElapsed, int64(bytes))
		}
	}
}
