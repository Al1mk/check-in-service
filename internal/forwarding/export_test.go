package forwarding

import "time"

// defaultRetryDelays holds the production schedule so SetRetryDelaysForTest
// can restore it after a test overrides it.
var defaultRetryDelays = retryDelays

// SetRetryDelaysForTest replaces the retry schedule for the duration of a test.
// Pass nil to restore the production defaults.
// Must not be called concurrently with RunWorker.
func SetRetryDelaysForTest(delays []time.Duration) {
	if delays == nil {
		retryDelays = defaultRetryDelays
	} else {
		retryDelays = delays
	}
}
