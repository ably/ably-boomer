package main

import (
	"log"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/ably/perf"
)

func taskFn(testConfig TestConfig) func() {
	switch testConfig.TestType {
	case "fanout":
		return curryFanOutTask(testConfig)
	case "personal":
		return curryPersonalTask(testConfig)
	case "sharded":
		return curryShardedTask(testConfig)
	default:
		panic("Unknown test type: '" + testConfig.TestType + "'")
	}
}

func main() {
	testConfig := newTestConfig()
	perf := perf.New()

	fn := taskFn(testConfig)

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
