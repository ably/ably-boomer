package ablyboomer

import (
	"context"
	"fmt"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/ably"
	"github.com/ably/ably-boomer/config"
	"github.com/inconshreveable/log15"
)

type TaskFn func(ctx context.Context)

type Task struct {
	Name string
	Fn   TaskFn
}

func RunAblyTask(ctx context.Context, config *config.Config, log log15.Logger) error {
	task := &Task{
		Name: config.TestType,
	}
	switch config.TestType {
	case "fanout":
		task.Fn = ably.FanOutTask(config, log)
	case "personal":
		task.Fn = ably.PersonalTask(config, log)
	case "sharded":
		task.Fn = ably.ShardedTask(config, log)
	case "composite":
		task.Fn = ably.CompositeTask(config, log)
	default:
		return fmt.Errorf("unknown Ably test type: %q", config.TestType)
	}
	return RunTask(ctx, config, task, log)
}

func RunTask(ctx context.Context, config *config.Config, task *Task, log log15.Logger) error {
	log.Info("setting Locust config", "host", config.LocustHost, "port", config.LocustPort, "0.9.0-support", config.Locust090)
	boomer.MasterHost = config.LocustHost
	boomer.MasterPort = config.LocustPort
	boomer.MasterVersionV090 = config.Locust090

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	boomer.Events.Subscribe("boomer:stop", cancel)

	log.Info("running boomer task", "name", task.Name)
	boomer.Run(&boomer.Task{
		Name: task.Name,
		Fn:   func() { task.Fn(ctx) },
	})
	log.Info("boomer task stopped", "name", task.Name)
	return nil
}
