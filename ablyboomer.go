package ablyboomer

import (
	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/ably"
	"github.com/ably/ably-boomer/config"
	"github.com/inconshreveable/log15"
)

func RunAblyTask(config *config.Config, log log15.Logger) error {
	task, err := ably.NewTask(config, log)
	if err != nil {
		return err
	}
	return RunTask(config, task, log)
}

func RunTask(config *config.Config, task *boomer.Task, log log15.Logger) error {
	log.Info("setting Locust config", "host", config.LocustHost, "port", config.LocustPort, "0.9.0-support", config.Locust090)
	boomer.MasterHost = config.LocustHost
	boomer.MasterPort = config.LocustPort
	boomer.MasterVersionV090 = config.Locust090

	log.Info("running boomer task", "name", task.Name)
	boomer.Run(task)
	log.Info("boomer task stopped", "name", task.Name)
	return nil
}
