package ably

import (
	"fmt"

	"github.com/ably/ably-boomer/config"
	"github.com/inconshreveable/log15"
)

func TaskFn(config *config.Config, log log15.Logger) (func(), error) {
	switch config.TestType {
	case "fanout":
		return curryFanOutTask(config, log), nil
	case "personal":
		return curryPersonalTask(config, log), nil
	case "sharded":
		return curryShardedTask(config, log), nil
	case "composite":
		return curryCompositeTask(config, log), nil
	default:
		return nil, fmt.Errorf("unknown Ably test type: %q", config.TestType)
	}
}
