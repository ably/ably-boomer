package main

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/ably-forks/boomer"
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

	fn := taskFn(testConfig)

	task := &boomer.Task{
		Name: testConfig.TestType,
		Fn:   fn,
	}

	if testConfig.CPUProfile != "" {
		f, err := os.Create(testConfig.CPUProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	boomer.Run(task)
}
