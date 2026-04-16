package attendance

import "time"

// CheckInRecord represents an open (ongoing) shift for one employee.
type CheckInRecord struct {
	FactoryID         string
	FactoryLocation   string    // IANA timezone, e.g. "Europe/Berlin"
	HardwareTimestamp time.Time // card-reader clock, stored in UTC — the business event time
	ServerTimestamp   time.Time // time.Now() at receipt — used as drift sanity reference
}

// ShiftRecord represents a completed shift.
type ShiftRecord struct {
	EmployeeID      string
	FactoryID       string
	FactoryLocation string    // IANA timezone used when this shift was recorded
	CheckInHW       time.Time // hardware timestamp of check-in (UTC)
	CheckOutHW      time.Time // hardware timestamp of check-out (UTC)
	Minutes         int       // shift duration in whole minutes
	WeekStart       time.Time // Monday 00:00:00 in factory-local time, stored as UTC
	                          // Defines the calendar week (Monday–Sunday) this shift belongs to.
}
