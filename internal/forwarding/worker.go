// Package forwarding sends completed shift data to the external labor cost system.
// It runs as a background goroutine, reading jobs from a channel and retrying
// failed deliveries with a fixed backoff schedule.
package forwarding

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Job represents one unit of work: forward a completed shift to the external system.
type Job struct {
	EmployeeID    string
	MinutesWorked int
}

// retryDelays defines the wait before each retry attempt.
// A job is tried once immediately, then retried len(retryDelays) times.
// Total maximum attempts = 1 + len(retryDelays) = 4.
var retryDelays = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

// defaultRetryDelays holds the original delays so SetRetryDelaysForTest can restore them.
var defaultRetryDelays = retryDelays

// SetRetryDelaysForTest replaces the retry schedule for testing.
// Pass nil to restore the production defaults.
// Must not be called concurrently with RunWorker.
func SetRetryDelaysForTest(delays []time.Duration) {
	if delays == nil {
		retryDelays = defaultRetryDelays
	} else {
		retryDelays = delays
	}
}

// RunWorker reads jobs from the channel and forwards each one to targetURL.
// On failure it waits according to retryDelays before retrying.
// After all retries are exhausted the job is logged and discarded.
//
// Blocks until jobs is closed. Intended to be launched as a goroutine:
//
//	go forwarding.RunWorker(jobs, "http://localhost:8080/mock/recording", log.Default())
func RunWorker(jobs <-chan Job, targetURL string, logger *log.Logger) {
	client := &http.Client{Timeout: 5 * time.Second}
	for job := range jobs {
		deliver(client, job, targetURL, logger)
	}
}

// deliver attempts to POST job to targetURL, retrying on failure.
func deliver(client *http.Client, job Job, targetURL string, logger *log.Logger) {
	body := fmt.Sprintf(`{"employee_id":%q,"minutes_worked":%d}`, job.EmployeeID, job.MinutesWorked)

	if tryPost(client, targetURL, body) {
		logger.Printf("forwarding: delivered employee=%s minutes=%d", job.EmployeeID, job.MinutesWorked)
		return
	}

	for i, delay := range retryDelays {
		logger.Printf("forwarding: attempt %d failed for employee=%s, retrying in %s", i+1, job.EmployeeID, delay)
		time.Sleep(delay)
		if tryPost(client, targetURL, body) {
			logger.Printf("forwarding: delivered employee=%s minutes=%d (attempt %d)", job.EmployeeID, job.MinutesWorked, i+2)
			return
		}
	}

	logger.Printf("forwarding: discarding job for employee=%s after %d attempts", job.EmployeeID, 1+len(retryDelays))
}

// tryPost posts body to targetURL and returns true on HTTP 2xx, false otherwise.
// Network errors and non-2xx responses are both treated as failures.
func tryPost(client *http.Client, targetURL, body string) bool {
	resp, err := client.Post(targetURL, "application/json", strings.NewReader(body))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
