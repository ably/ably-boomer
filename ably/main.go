package main

import (
	"github.com/ably-forks/boomer"
)

func taskFn(testConfig TestConfig) func() {
	switch testConfig.TestType {
	case "fanout":
		return curryFanOutTask(testConfig)
	case "personal":
		return curryPersonalTask(testConfig)
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

	boomer.Run(task)
}
