package main

import "log"

func demoChannelNoResult(tasks []task) {
	done := make(chan struct{}, len(tasks))

	for _, currentTask := range tasks {
		go func() {
			err := runTask(currentTask)
			if err != nil {
				log.Printf("task %s failed with error: %v", currentTask.name, err)
			}
			done <- struct{}{}
		}()
	}

	for range tasks {
		<-done
	}
}
