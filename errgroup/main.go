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
	demo3Errgroup(tasks)

	errorTasks := sampleTasksMultipleFailures()

	fmt.Println("\nerrgroup.Group fail-fast:")
	demo3ErrgroupFailFastCollect(errorTasks)

	fmt.Println("\nerrgroup.Group, collecting all errors and results:")
	demo3ErrgroupCollectAllWithResults(tasks)
}
