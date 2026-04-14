package main

import (
	"log"
	"net/http"
)

func main() {
	bff, err := newBFFApp()
	if err != nil {
		log.Fatalf("bff backend setup failed: %v", err)
	}
	defer func() {
		if closeErr := bff.traceLogger.Close(); closeErr != nil {
			log.Printf("close bff trace logger: %v", closeErr)
		}
	}()

	log.Printf("combined bff backend and resource api listening on http://localhost:8082")
	if err := http.ListenAndServe(":8082", bff.routes()); err != nil {
		log.Fatalf("bff backend failed: %v", err)
	}
}
