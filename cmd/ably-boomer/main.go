package main

import (
	"os"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/perf"
	"github.com/ably/ably-boomer/tasks/ably"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2"
)

var log = log15.New()

func run(taskFn func(), c *cli.Context) error {
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

func runComposite(c *cli.Context) error {
	apiKey := c.String(apiKeyFlag.Name)
	env := c.String(envFlag.Name)
	channelName := c.String(channelNameFlag.Name)
	numChannels := c.Int(numChannelsFlag.Name)
	msgDataLength := c.Int(msgDataLengthFlag.Name)
	numSubscriptions := c.Int(numSubscriptionsFlag.Name)
	publishInterval := c.Int(publishIntervalFlag.Name)
	taskFn := ably.CurryCompositeTask(ably.CompositeConf{
		Logger:           log,
		APIKey:           apiKey,
		Env:              env,
		ChannelName:      channelName,
		NumChannels:      numChannels,
		MsgDataLength:    msgDataLength,
		NumSubscriptions: numSubscriptions,
		PublishInterval:  publishInterval,
	})

	return run(taskFn, c)
}

func main() {
	log.Info("initialising ably-boomer")

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "composite",
				Aliases: []string{"c"},
				Usage:   "run a composite task",
				Action:  runComposite,
				Flags: []cli.Flag{
					apiKeyFlag,
					envFlag,
					channelNameFlag,
					numChannelsFlag,
					msgDataLengthFlag,
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
