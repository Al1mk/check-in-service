package main

import (
	"log"
	"net/http"

	"github.com/Al1mk/check-in-service/internal/attendance"
	"github.com/Al1mk/check-in-service/internal/httpapi"
	"github.com/Al1mk/check-in-service/internal/mock"
)

func main() {
	store := attendance.NewStore()

	mux := http.NewServeMux()
	mux.Handle("POST /events", httpapi.NewEventHandler(store))
	mux.Handle("POST /mock/recording", mock.NewRecordingHandler())

	// Forwarding worker wired here in Phase 4.

	addr := ":8080"
	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
