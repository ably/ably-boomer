package main

import (
	"os"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/ably/perf"
	"github.com/inconshreveable/log15"
)

var log = log15.New()

func taskFn(testConfig TestConfig) func() {
	switch testConfig.TestType {
	case "fanout":
		return curryFanOutTask(testConfig)
	case "personal":
		return curryPersonalTask(testConfig)
	case "sharded":
		return curryShardedTask(testConfig)
	case "composite":
		return curryCompositeTask(testConfig)
	default:
		panic("Unknown test type: '" + testConfig.TestType + "'")
	}
}

func main() {
	log.Info("initialising ably-boomer")
	conf := newTestConfig()

	task := &boomer.Task{
		Name: conf.TestType,
		Fn:   taskFn(conf),
	}

	log.Info("starting perf")
	perf := perf.New()
	if err := perf.Start(); err != nil {
		log.Crit("error starting perf", "err", err)
		os.Exit(1)
	}
	defer perf.Stop()

	log.Info("running ably-boomer", "test-type", conf.TestType)
	boomer.Run(task)
}
