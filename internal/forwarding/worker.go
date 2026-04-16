// Package forwarding sends completed shift data to the external labor cost system.
// It runs as a background goroutine, reading jobs from a channel and retrying
// failed deliveries with exponential backoff.
package forwarding

// Job represents one unit of work: forward a completed shift to the external system.
type Job struct {
	EmployeeID    string
	MinutesWorked int
	Attempt       int
}

// RunWorker reads jobs from the channel and forwards them to the external system.
// It retries on failure using exponential backoff and discards after MaxRetries.
// Intended to be launched as a goroutine: go forwarding.RunWorker(jobs)
func RunWorker(jobs <-chan Job) {
	// TODO: implement
}
