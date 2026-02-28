package main

import (
	"log"
	"sync"
)

func demoWaitGroupNoResult(tasks []task) {
	var wg sync.WaitGroup

	for _, currentTask := range tasks {
		wg.Go(func() {
			err := runTask(currentTask)
			if err != nil {
				log.Printf("task %s failed with error: %v", currentTask.name, err)
			}
		})
	}

	wg.Wait()
}
