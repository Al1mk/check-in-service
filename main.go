package main

import (
	"log"
	"net/http"

	"github.com/Al1mk/check-in-service/internal/attendance"
	"github.com/Al1mk/check-in-service/internal/forwarding"
	"github.com/Al1mk/check-in-service/internal/httpapi"
	"github.com/Al1mk/check-in-service/internal/mock"
)

func main() {
	store := attendance.NewStore()
	jobs := make(chan forwarding.Job, 100)

	mux := http.NewServeMux()
	mux.Handle("POST /events", httpapi.NewEventHandler(store, jobs))
	mux.Handle("POST /mock/recording", mock.NewRecordingHandler())

	// Worker and server are started here once their implementations are complete.
	// go forwarding.RunWorker(jobs)

	addr := ":8080"
	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
