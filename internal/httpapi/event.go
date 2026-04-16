// Package httpapi contains the HTTP handlers and transport types for the check-in service.
package httpapi

import (
	"net/http"

	"github.com/Al1mk/check-in-service/internal/attendance"
	"github.com/Al1mk/check-in-service/internal/forwarding"
)

// EventHandler handles POST /events.
type EventHandler struct {
	store *attendance.Store
	jobs  chan<- forwarding.Job
}

// NewEventHandler constructs an EventHandler wired to the given store and job channel.
func NewEventHandler(store *attendance.Store, jobs chan<- forwarding.Job) *EventHandler {
	return &EventHandler{store: store, jobs: jobs}
}

// ServeHTTP validates the incoming event, updates attendance state, and on check-out
// enqueues a forwarding job without blocking the HTTP response.
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
