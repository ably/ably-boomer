package main

import (
	"log"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/ably/perf"
)

func taskFn(testConfig TestConfig, perf *perf.Reporter) func() {
	switch testConfig.TestType {
	case "fanout":
		return curryFanOutTask(testConfig, perf)
	case "personal":
		return curryPersonalTask(testConfig, perf)
	case "sharded":
		return curryShardedTask(testConfig, perf)
	default:
		panic("Unknown test type: '" + testConfig.TestType + "'")
	}
}

func main() {
	testConfig := newTestConfig()
	perf := perf.NewReporter()

	fn := taskFn(testConfig, perf)

	task := &boomer.Task{
		Name: testConfig.TestType,
		Fn:   fn,
	}

	perfError := perf.Start()
	if perfError != nil {
		log.Fatalf("errror starting perf: %s", perfError)
	}
	defer perf.Stop()

	boomer.Run(task)
}
