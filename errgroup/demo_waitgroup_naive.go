package main

import (
	"log"
	"sync"
)

func demoWaitGroupWithResult(tasks []task) {
	var wg sync.WaitGroup
	results := make([]string, len(tasks))

	for i, currentTask := range tasks {
		wg.Go(func() {
			data, err := runTaskWithResult(currentTask)
			if err != nil {
				log.Printf("task %s failed with error: %v", currentTask.name, err)
			} else {
				results[i] = data
			}
		})
	}
	wg.Wait()

	for _, result := range results {
		if result != "" {
			log.Printf("result: %s", result)
		}
	}

}
