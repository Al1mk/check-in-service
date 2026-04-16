package attendance

import "errors"

// Sentinel errors returned by Store methods.
// Callers use errors.Is to distinguish them.
var (
	// ErrAlreadyCheckedIn is returned when CheckIn is called for an employee
	// who already has an active open shift. The caller must treat this as a
	// conflict; no automatic repair is performed.
	//
	// Reliable duplicate suppression (card reader retries sending the same
	// event twice) requires an idempotency key or event ID from the device.
	// Without one, the service cannot distinguish a retry from a second
	// genuine check-in attempt, so the second request is always rejected.
	ErrAlreadyCheckedIn = errors.New("employee already checked in")

	// ErrNotCheckedIn is returned when CheckOut is called for an employee
	// who has no active shift.
	ErrNotCheckedIn = errors.New("employee not checked in")

	// ErrCheckOutBeforeCheckIn is returned when the check-out hardware
	// timestamp is at or before the check-in hardware timestamp.
	ErrCheckOutBeforeCheckIn = errors.New("check-out time is not after check-in time")

	// ErrClockDrift is returned when the hardware timestamp deviates from the
	// server receipt time by more than MaxClockDrift.
	//
	// Exercise assumption: the 5-minute threshold is a named constant chosen
	// as a reasonable default for factory environments with reliable NTP. In
	// production this should be an environment variable, and a spike in drift
	// rejections should trigger an alert for a misbehaving card reader.
	ErrClockDrift = errors.New("hardware timestamp deviates from server time beyond allowed drift")

	// ErrUnknownTimezone is returned when a factory location string cannot be
	// loaded as a valid IANA timezone.
	ErrUnknownTimezone = errors.New("unknown or invalid factory timezone")
)
