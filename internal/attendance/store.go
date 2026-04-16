// Package attendance tracks employee check-in/out state and shift history.
package attendance

import (
	"sync"
	"time"
)

// MaxClockDrift is the maximum tolerated difference between a card reader's
// hardware timestamp and the server receipt time.
//
// Exercise assumption: 5 minutes is a reasonable default for factory
// environments with reliable NTP. In production this should be an environment
// variable; a spike in drift rejections indicates a card reader with a broken
// or drifting clock and should trigger an alert.
const MaxClockDrift = 5 * time.Minute

// Store holds the in-memory attendance state for all employees.
// It is safe for concurrent use.
type Store struct {
	mu      sync.RWMutex
	active  map[string]*CheckInRecord // employeeID → open shift
	history map[string][]ShiftRecord  // employeeID → closed shifts
}

// NewStore returns an empty, ready-to-use Store.
func NewStore() *Store {
	return &Store{
		active:  make(map[string]*CheckInRecord),
		history: make(map[string][]ShiftRecord),
	}
}

// CheckIn records an employee's arrival at a factory.
//
// hwTime is the card-reader clock timestamp — the business event time.
// serverTime is time.Now() at receipt, used only as a drift sanity check.
//
// Returns ErrAlreadyCheckedIn if the employee already has an active shift.
// No automatic repair is performed; missed check-outs must be resolved
// through an administrative process (see README).
//
// Returns ErrClockDrift if hwTime deviates from serverTime by more than
// MaxClockDrift.
//
// Returns ErrUnknownTimezone if factoryLocation is not a valid IANA name.
func (s *Store) CheckIn(employeeID, factoryID, factoryLocation string, hwTime, serverTime time.Time) error {
	if _, err := time.LoadLocation(factoryLocation); err != nil {
		return ErrUnknownTimezone
	}
	if err := validateDrift(hwTime, serverTime); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.active[employeeID]; ok {
		return ErrAlreadyCheckedIn
	}

	s.active[employeeID] = &CheckInRecord{
		FactoryID:         factoryID,
		FactoryLocation:   factoryLocation,
		HardwareTimestamp: hwTime.UTC(),
		ServerTimestamp:   serverTime.UTC(),
	}
	return nil
}

// CheckOut records an employee's departure and closes the active shift.
//
// Returns the completed ShiftRecord and the employee's total minutes worked
// in the current calendar week (Monday 00:00 – Sunday 23:59 in the factory's
// local timezone), both computed atomically under the same lock.
//
// Returns ErrNotCheckedIn if the employee has no active shift.
// Returns ErrCheckOutBeforeCheckIn if hwTime is not after the check-in hwTime.
// Returns ErrClockDrift if hwTime deviates from serverTime by more than MaxClockDrift.
func (s *Store) CheckOut(employeeID string, hwTime, serverTime time.Time) (ShiftRecord, int, error) {
	if err := validateDrift(hwTime, serverTime); err != nil {
		return ShiftRecord{}, 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.active[employeeID]
	if !ok {
		return ShiftRecord{}, 0, ErrNotCheckedIn
	}
	if !hwTime.After(record.HardwareTimestamp) {
		return ShiftRecord{}, 0, ErrCheckOutBeforeCheckIn
	}

	shift, err := closeShift(employeeID, record, hwTime)
	if err != nil {
		return ShiftRecord{}, 0, err
	}

	s.history[employeeID] = append(s.history[employeeID], shift)
	delete(s.active, employeeID)

	weekTotal := sumWeekMinutes(s.history[employeeID], shift.WeekStart)
	return shift, weekTotal, nil
}

// WeeklyMinutes returns the total minutes worked by an employee in the
// Monday–Sunday calendar week that contains ref, expressed in the given
// IANA timezone. It does not include any currently open shift.
func (s *Store) WeeklyMinutes(employeeID string, ref time.Time, timezone string) (int, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return 0, ErrUnknownTimezone
	}
	weekStart := mondayMidnight(ref, loc)

	s.mu.RLock()
	defer s.mu.RUnlock()
	return sumWeekMinutes(s.history[employeeID], weekStart), nil
}

// --- helpers (unexported) ---

// validateDrift returns ErrClockDrift if hwTime and serverTime differ by more
// than MaxClockDrift.
func validateDrift(hwTime, serverTime time.Time) error {
	diff := hwTime.Sub(serverTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > MaxClockDrift {
		return ErrClockDrift
	}
	return nil
}

// closeShift builds a ShiftRecord from an open CheckInRecord and a check-out
// hardware timestamp. The WeekStart field is the Monday 00:00:00 of the week
// that contains the check-in, expressed in factory-local time (stored as UTC).
func closeShift(employeeID string, rec *CheckInRecord, checkOutHW time.Time) (ShiftRecord, error) {
	loc, err := time.LoadLocation(rec.FactoryLocation)
	if err != nil {
		return ShiftRecord{}, ErrUnknownTimezone
	}
	minutes := int(checkOutHW.UTC().Sub(rec.HardwareTimestamp).Minutes())
	weekStart := mondayMidnight(rec.HardwareTimestamp, loc)
	return ShiftRecord{
		EmployeeID:      employeeID,
		FactoryID:       rec.FactoryID,
		FactoryLocation: rec.FactoryLocation,
		CheckInHW:       rec.HardwareTimestamp,
		CheckOutHW:      checkOutHW.UTC(),
		Minutes:         minutes,
		WeekStart:       weekStart,
	}, nil
}

// mondayMidnight returns the Monday 00:00:00 of the calendar week containing t,
// expressed in loc and then converted to UTC for storage.
// ISO calendar week: Monday is day 1, Sunday is day 7.
func mondayMidnight(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	// time.Weekday(): Sunday=0, Monday=1, …, Saturday=6
	// Days since the most recent Monday:
	daysSinceMonday := int(local.Weekday()+6) % 7 // Sun→6, Mon→0, Tue→1, …
	monday := local.AddDate(0, 0, -daysSinceMonday)
	// Truncate to midnight in local time.
	y, m, d := monday.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc).UTC()
}

// sumWeekMinutes sums the Minutes of all ShiftRecords whose WeekStart equals
// the given weekStart (compared as UTC instants).
func sumWeekMinutes(shifts []ShiftRecord, weekStart time.Time) int {
	total := 0
	for _, s := range shifts {
		if s.WeekStart.Equal(weekStart) {
			total += s.Minutes
		}
	}
	return total
}
