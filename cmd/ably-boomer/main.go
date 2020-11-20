package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-boomer/tasks"
	"github.com/ably/ably-boomer/tasks/ably"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2"
)

var log = log15.New()

func ablyAction(c *cli.Context) error {
	return tasks.RunWithBoomer(log, ably.TaskFn(log, config.ParseConf(c)), c)
}

func main() {
	log.Info("initialising ably-boomer")

	rand.Seed(time.Now().UnixNano())

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "ably",
				Aliases: []string{"a"},
				Usage:   "run an Ably Runtime task",
				Action:  ablyAction,
				Flags:   config.TaskFlags,
			},
		},
		Name:  "ably-boomer",
		Usage: "Ably load generator for Locust, based on the boomer library",
		CommandNotFound: func(c *cli.Context, comm string) {
			log.Crit("command not found", "command", comm)
		},
		Flags: config.CommonFlags,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("fatal error", "err", err)
		os.Exit(1)
	}
}
