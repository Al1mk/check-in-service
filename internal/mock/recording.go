// Package mock provides a fake external labor cost system for local testing.
// It randomly fails or delays to exercise the forwarding worker's retry logic.
package mock

import (
	"math/rand"
	"net/http"
	"time"
)

// RecordingHandler handles POST /mock/recording.
//
// Approximate response distribution:
//   - 30 % → 500 Internal Server Error immediately
//   - 20 % → 200 OK after a random delay (delayMin – delayMax)
//   - 50 % → 200 OK immediately
//
// In production the delay range is 2–5 seconds.
// Use NewRecordingHandlerWithDelay(0, 0) in tests to suppress the slow path.
type RecordingHandler struct {
	delayMin time.Duration
	delayMax time.Duration
}

// NewRecordingHandler returns a handler with the production delay range (2–5 s).
func NewRecordingHandler() *RecordingHandler {
	return &RecordingHandler{delayMin: 2 * time.Second, delayMax: 5 * time.Second}
}

// NewRecordingHandlerWithDelay returns a handler with a configurable delay range.
// Pass (0, 0) to disable delays entirely.
func NewRecordingHandlerWithDelay(min, max time.Duration) *RecordingHandler {
	return &RecordingHandler{delayMin: min, delayMax: max}
}

// ServeHTTP implements http.Handler.
func (h *RecordingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n := rand.Intn(10) // 0–9

	switch {
	case n < 3: // 30 %: immediate failure
		http.Error(w, "simulated failure", http.StatusInternalServerError)
	case n < 5: // 20 %: slow success
		if h.delayMax > 0 {
			spread := int64(h.delayMax-h.delayMin) + 1
			delay := h.delayMin + time.Duration(rand.Int63n(spread))
			time.Sleep(delay)
		}
		w.WriteHeader(http.StatusOK)
	default: // 50 %: immediate success
		w.WriteHeader(http.StatusOK)
	}
}
