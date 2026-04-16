// Package mock provides a fake external labor cost system for local testing.
// It randomly fails or delays to exercise the forwarding worker's retry logic.
package mock

import "net/http"

// RecordingHandler handles POST /mock/recording.
// Behaviour (approximate distribution):
//   - 30 % of requests → HTTP 500 immediately
//   - 20 % of requests → random delay (2–5 s), then HTTP 200
//   - 50 % of requests → HTTP 200 immediately
type RecordingHandler struct{}

// NewRecordingHandler returns a ready-to-register RecordingHandler.
func NewRecordingHandler() *RecordingHandler {
	return &RecordingHandler{}
}

// ServeHTTP implements http.Handler.
func (h *RecordingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
