package ably

import (
	"fmt"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/config"
	"github.com/inconshreveable/log15"
)

func NewTask(config *config.Config, log log15.Logger) (*boomer.Task, error) {
	var taskFn func()
	switch config.TestType {
	case "fanout":
		taskFn = curryFanOutTask(config, log)
	case "personal":
		taskFn = curryPersonalTask(config, log)
	case "sharded":
		taskFn = curryShardedTask(config, log)
	case "composite":
		taskFn = curryCompositeTask(config, log)
	default:
		return nil, fmt.Errorf("unknown Ably test type: %q", config.TestType)
	}
	return &boomer.Task{
		Name: config.TestType,
		Fn:   taskFn,
	}, nil
}
