package main

import (
	"fmt"
	"os"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/perf"
	"github.com/ably/ably-boomer/tasks/ably"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2"
)

var log = log15.New()

func taskFn(c *cli.Context) (func(), error) {
	testType := c.String(testTypeFlag.Name)
	switch testType {
	case "fanout":
		apiKey := c.String(apiKeyFlag.Name)
		env := c.String(envFlag.Name)
		channelName := c.String(channelNameFlag.Name)
		return ably.CurryFanOutTask(ably.FanOutConf{
			Logger:      log,
			APIKey:      apiKey,
			Env:         env,
			ChannelName: channelName,
		}), nil
	case "personal":
		apiKey := c.String(apiKeyFlag.Name)
		env := c.String(envFlag.Name)
		publishInterval := c.Int(publishIntervalFlag.Name)
		numSubscriptions := c.Int(numSubscriptionsFlag.Name)
		msgDataLength := c.Int(msgDataLengthFlag.Name)
		return ably.CurryPersonalTask(ably.PersonalConf{
			Logger:           log,
			APIKey:           apiKey,
			Env:              env,
			PublishInterval:  publishInterval,
			NumSubscriptions: numSubscriptions,
			MsgDataLength:    msgDataLength,
		}), nil
	case "sharded":
		apiKey := c.String(apiKeyFlag.Name)
		env := c.String(envFlag.Name)
		numChannels := c.Int(numChannelsFlag.Name)
		publishInterval := c.Int(publishIntervalFlag.Name)
		msgDataLength := c.Int(msgDataLengthFlag.Name)
		numSubscriptions := c.Int(numSubscriptionsFlag.Name)
		publisher := c.Bool(publisherFlag.Name)
		return ably.CurryShardedTask(ably.ShardedConf{
			Logger:           log,
			APIKey:           apiKey,
			Env:              env,
			NumChannels:      numChannels,
			PublishInterval:  publishInterval,
			MsgDataLength:    msgDataLength,
			NumSubscriptions: numSubscriptions,
			Publisher:        publisher,
		}), nil
	case "composite":
		apiKey := c.String(apiKeyFlag.Name)
		env := c.String(envFlag.Name)
		channelName := c.String(channelNameFlag.Name)
		numChannels := c.Int(numChannelsFlag.Name)
		msgDataLength := c.Int(msgDataLengthFlag.Name)
		numSubscriptions := c.Int(numSubscriptionsFlag.Name)
		publishInterval := c.Int(publishIntervalFlag.Name)
		return ably.CurryCompositeTask(ably.CompositeConf{
			Logger:           log,
			APIKey:           apiKey,
			Env:              env,
			ChannelName:      channelName,
			NumChannels:      numChannels,
			MsgDataLength:    msgDataLength,
			NumSubscriptions: numSubscriptions,
			PublishInterval:  publishInterval,
		}), nil
	default:
		return nil, fmt.Errorf("unknown test type: %s", testType)
	}
}

func run(c *cli.Context) error {
	fn, err := taskFn(c)
	if err != nil {
		return nil
	}

	testType := c.String(testTypeFlag.Name)
	task := &boomer.Task{
		Name: testType,
		Fn:   fn,
	}

	log.Info("starting perf")
	perf := perf.New(perf.Conf{
		CPUProfileDir: c.Path(cpuProfileDirFlag.Name),
		S3Bucket:      c.String(s3BucketFlag.Name),
	})
	if err = perf.Start(); err != nil {
		log.Crit("error starting perf", "err", err)
		os.Exit(1)
	}
	defer perf.Stop()

	log.Info("running ably-boomer", "test-type", testType)
	boomer.Run(task)

	return nil
}

func main() {
	log.Info("initialising ably-boomer")

	ablyFlags := []cli.Flag{
		testTypeFlag,
		envFlag,
		apiKeyFlag,
		channelNameFlag,
		publishIntervalFlag,
		numSubscriptionsFlag,
		msgDataLengthFlag,
		publisherFlag,
		numChannelsFlag,
	}
	perfFlags := []cli.Flag{
		cpuProfileDirFlag,
		s3BucketFlag,
	}
	awsFlags := []cli.Flag{
		regionFlag,
		sdkLoadConfigFlag,
		profileFlag,
		accessKeyIDFlag,
		secretAccessKeyFlag,
		sessionTokenFlag,
	}

	app := &cli.App{
		Flags:  append(append(ablyFlags, perfFlags...), awsFlags...),
		Name:   "ably-boomer",
		Usage:  "Ably load generator for Locust, based on the boomer library",
		Action: run,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("fatal error", "err", err)
		os.Exit(1)
	}
}
