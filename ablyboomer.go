package ablyboomer

import (
	"github.com/ably-forks/boomer"
	"github.com/inconshreveable/log15"
)

func Run(taskName string, taskFn func(), log log15.Logger) error {
	log.Info("running ably-boomer", "task", taskName)
	boomer.Run(&boomer.Task{
		Name: taskName,
		Fn:   taskFn,
	})
	log.Info("ably-boomer stopped", "task", taskName)
	return nil
}
