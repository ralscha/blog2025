package main

import (
	"fmt"
	"log"
)

func main() {
	log.SetFlags(0)

	tasks := sampleTasks()

	fmt.Println("sync.WaitGroup with no result:")
	demoWaitGroupNoResult(tasks)

	fmt.Println("\nsync.WaitGroup with result:")
	demoWaitGroupWithResult(tasks)

	fmt.Println("\nerrgroup.Group:")
	demoErrgroup(tasks)

	errorTasks := sampleTasksMultipleFailures()

	fmt.Println("\nerrgroup.Group fail-fast:")
	demoErrgroupFailFastCollect(errorTasks)

	fmt.Println("\nerrgroup.Group, collecting all errors and results:")
	demoErrgroupCollectAllWithResults(tasks)
}
