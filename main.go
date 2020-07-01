package main

import (
	"github.com/ably-forks/boomer"
)

func taskFn() func() {
	testType := ablyTestType()

	switch testType {
	case "fanout":
		return curryFanOutTask()
	default:
		panic("Unknown test type: '" + testType + "'")
	}
}

func main() {
	task := &boomer.Task{
		Name: ablyTestType(),
		Fn:   taskFn(),
	}

	boomer.Run(task)
}
