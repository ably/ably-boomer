package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/perf"
	"github.com/ably/ably-boomer/tasks"
	"github.com/ably/ably-boomer/tasks/ably"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2"
)

var log = log15.New()

func runWithBoomer(taskFn func(), c *cli.Context) error {
	taskName := c.Command.Name

	task := &boomer.Task{
		Name: taskName,
		Fn:   taskFn,
	}

	log.Info("starting perf")
	perf := perf.New(perf.Conf{
		CPUProfileDir: c.Path(cpuProfileDirFlag.Name),
		S3Bucket:      c.String(s3BucketFlag.Name),
	})
	if err := perf.Start(); err != nil {
		log.Crit("error starting perf", "err", err)
		os.Exit(1)
	}
	defer perf.Stop()

	log.Info("running ably-boomer", "task name", taskName)
	args := c.StringSlice(boomerArgsFlag.Name)
	os.Args = append([]string{"boomer"}, args...)
	boomer.Run(task)

	return nil
}

func ablyAction(c *cli.Context) error {
	return runWithBoomer(ably.TaskFn(log, parseConf(c)), c)
}

func parseConf(c *cli.Context) tasks.Conf {
	return tasks.Conf{
		APIKey:           c.String(apiKeyFlag.Name),
		Env:              c.String(envFlag.Name),
		NumChannels:      c.Int(numChannelsFlag.Name),
		MsgDataLength:    c.Int(msgDataLengthFlag.Name),
		SSESubscriber:    c.Bool(sseSubscriberFlag.Name),
		NumSubscriptions: c.Int(numSubscriptionsFlag.Name),
		PublishInterval:  c.Int(publishIntervalFlag.Name),
	}
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
				Flags: []cli.Flag{
					apiKeyFlag,
					envFlag,
					numChannelsFlag,
					msgDataLengthFlag,
					sseSubscriberFlag,
					numSubscriptionsFlag,
					publishIntervalFlag,
				},
			},
		},
		Name:  "ably-boomer",
		Usage: "Ably load generator for Locust, based on the boomer library",
		CommandNotFound: func(c *cli.Context, comm string) {
			log.Crit("command not found", "command", comm)
		},
		Flags: []cli.Flag{boomerArgsFlag},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("fatal error", "err", err)
		os.Exit(1)
	}
}
