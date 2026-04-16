package forwarding_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Al1mk/check-in-service/internal/forwarding"
)

// discardLogger suppresses log output during tests.
var discardLogger = log.New(io.Discard, "", 0)

// runJob sends one job through a fresh channel to a worker pointed at srv,
// then waits up to timeout for the worker to consume and process it.
func runJob(t *testing.T, srv *httptest.Server, job forwarding.Job, timeout time.Duration) {
	t.Helper()
	jobs := make(chan forwarding.Job, 1)
	done := make(chan struct{})
	go func() {
		forwarding.RunWorker(jobs, srv.URL, discardLogger)
		close(done)
	}()
	jobs <- job
	close(jobs) // signal worker to stop after this job
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("worker did not finish within timeout")
	}
}

// TestWorker_DeliverOnFirstAttempt verifies that the worker posts to the target
// and succeeds without retrying when the server always returns 200.
func TestWorker_DeliverOnFirstAttempt(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	runJob(t, srv, forwarding.Job{EmployeeID: "E001", MinutesWorked: 480}, 5*time.Second)

	if n := hits.Load(); n != 1 {
		t.Errorf("expected 1 request to target, got %d", n)
	}
}

// TestWorker_RetriesOnFailure verifies that a job is retried when the server
// fails for the first two attempts and succeeds on the third.
func TestWorker_RetriesOnFailure(t *testing.T) {
	// retryDelays in the worker are [1s, 2s, 4s] in production, but the worker
	// package does not expose them. We verify retry count, not timing.
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Override retry delays to zero for fast tests.
	forwarding.SetRetryDelaysForTest([]time.Duration{0, 0, 0})
	defer forwarding.SetRetryDelaysForTest(nil) // restore defaults

	runJob(t, srv, forwarding.Job{EmployeeID: "E002", MinutesWorked: 60}, 5*time.Second)

	if n := hits.Load(); n != 3 {
		t.Errorf("expected 3 requests (2 failures + 1 success), got %d", n)
	}
}

// TestWorker_DiscardsAfterMaxRetries verifies that the worker stops after
// exhausting all retries when the server always fails.
func TestWorker_DiscardsAfterMaxRetries(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	forwarding.SetRetryDelaysForTest([]time.Duration{0, 0, 0})
	defer forwarding.SetRetryDelaysForTest(nil)

	runJob(t, srv, forwarding.Job{EmployeeID: "E003", MinutesWorked: 120}, 5*time.Second)

	// 1 initial attempt + 3 retries = 4 total
	if n := hits.Load(); n != 4 {
		t.Errorf("expected 4 requests (1 + 3 retries), got %d", n)
	}
}
