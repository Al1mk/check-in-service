package mock_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Al1mk/check-in-service/internal/mock"
)

// newTestHandler returns a RecordingHandler with delays suppressed so the
// probabilistic behaviour can be sampled quickly.
func newTestHandler() *mock.RecordingHandler {
	return mock.NewRecordingHandlerWithDelay(0, 0)
}

// TestRecordingHandler_ResponseCodes verifies that the handler only ever
// returns 200 or 500 — never any other status — across many requests.
func TestRecordingHandler_ResponseCodes(t *testing.T) {
	h := newTestHandler()
	const requests = 200

	for i := 0; i < requests; i++ {
		r := httptest.NewRequest(http.MethodPost, "/mock/recording", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		code := w.Code
		if code != http.StatusOK && code != http.StatusInternalServerError {
			t.Errorf("request %d: unexpected status %d", i, code)
		}
	}
}

// TestRecordingHandler_BothOutcomesOccur verifies that across enough requests
// the handler produces at least one success and at least one failure, confirming
// both branches are reachable.
func TestRecordingHandler_BothOutcomesOccur(t *testing.T) {
	h := newTestHandler()
	const requests = 100

	var ok, fail int
	for i := 0; i < requests; i++ {
		r := httptest.NewRequest(http.MethodPost, "/mock/recording", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			ok++
		} else {
			fail++
		}
	}

	if ok == 0 {
		t.Error("expected at least one 200 response across 100 requests")
	}
	if fail == 0 {
		t.Error("expected at least one 500 response across 100 requests")
	}
}
