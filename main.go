package main

import (
	"log"
	"net/http"

	"github.com/Al1mk/check-in-service/internal/attendance"
	"github.com/Al1mk/check-in-service/internal/forwarding"
	"github.com/Al1mk/check-in-service/internal/httpapi"
	"github.com/Al1mk/check-in-service/internal/mock"
)

const (
	addr           = ":8080"
	mockRecordingURL = "http://localhost:8080/mock/recording"
	jobQueueSize   = 100
)

func main() {
	logger := log.Default()
	store := attendance.NewStore()
	jobs := make(chan forwarding.Job, jobQueueSize)

	mux := http.NewServeMux()
	mux.Handle("POST /events", httpapi.NewEventHandler(store, jobs, logger))
	mux.Handle("POST /mock/recording", mock.NewRecordingHandler())

	go forwarding.RunWorker(jobs, mockRecordingURL, logger)

	logger.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
