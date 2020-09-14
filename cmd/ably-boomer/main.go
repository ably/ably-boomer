package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/perf"
	"github.com/ably/ably-boomer/tasks"
	"github.com/urfave/cli/v2"
)

func nameToEnvVars(name, prefix string) []string {
	envVar := strings.ToUpper(name)
	envVar = strings.ReplaceAll(envVar, "-", "_")
	envVar = fmt.Sprintf("%s_%s", prefix, envVar)
	return []string{envVar}
}

func generateEnvVars(prefix string, flags []cli.Flag) {
	for _, flag := range flags {
		switch f := flag.(type) {
		case *cli.BoolFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.DurationFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.Float64Flag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.Float64SliceFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.GenericFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.Int64Flag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.Int64SliceFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.IntFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.IntSliceFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.PathFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.StringFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.StringSliceFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.TimestampFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.Uint64Flag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		case *cli.UintFlag:
			f.EnvVars = nameToEnvVars(f.Name, prefix)
		}
	}
}

func taskFn(c *cli.Context) (func(), error) {
	testType := c.String(testTypeFlag.Name)
	switch testType {
	case "fanout":
		apiKey := c.String(apiKeyFlag.Name)
		env := c.String(envFlag.Name)
		channelName := c.String(channelNameFlag.Name)
		return tasks.CurryFanOutTask(tasks.FanOutConf{
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
		return tasks.CurryPersonalTask(tasks.PersonalConf{
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
		return tasks.CurryShardedTask(tasks.ShardedConf{
			APIKey:           apiKey,
			Env:              env,
			NumChannels:      numChannels,
			PublishInterval:  publishInterval,
			MsgDataLength:    msgDataLength,
			NumSubscriptions: numSubscriptions,
			Publisher:        publisher,
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

	perf := perf.New(perf.Conf{
		CPUProfileDir: c.Path(cpuProfileDirFlag.Name),
		S3Bucket:      c.String(s3BucketFlag.Name),
	})
	err = perf.Start()
	if err != nil {
		return fmt.Errorf("error starting perf: %w", err)
	}
	defer perf.Stop()

	boomer.Run(task)

	return nil
}

func main() {
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

	generateEnvVars("ABLY", ablyFlags)
	generateEnvVars("PERF", perfFlags)
	generateEnvVars("AWS", awsFlags)

	app := &cli.App{
		Flags:  append(append(ablyFlags, perfFlags...), awsFlags...),
		Name:   "ably-boomer",
		Usage:  "Ably load generator for Locust, based on the boomer library",
		Action: run,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
