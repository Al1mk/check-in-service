package attendance_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Al1mk/check-in-service/internal/attendance"
)

// fixed parses a UTC RFC3339 string into a time.Time. Panics on bad input so
// test data errors surface immediately rather than silently as zero values.
func fixed(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// server returns a server-receipt time 10s after hwTime so the drift guard
// passes.
func server(hwTime time.Time) time.Time {
	return hwTime.Add(10 * time.Second)
}

const (
	emp     = "E001"
	factory = "F-Munich-01"
	tzBerlin = "Europe/Berlin" // UTC+2 in summer (CEST)
)

// --- CheckIn ---

func TestCheckIn_Normal(t *testing.T) {
	s := attendance.NewStore()
	hw := fixed("2026-04-07T08:00:00Z")

	err := s.CheckIn(emp, factory, tzBerlin, hw, server(hw))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCheckIn_AlreadyCheckedIn_ReturnsError(t *testing.T) {
	s := attendance.NewStore()
	hw1 := fixed("2026-04-07T08:00:00Z")
	hw2 := fixed("2026-04-07T08:00:01Z") // one second later — still a conflict

	_ = s.CheckIn(emp, factory, tzBerlin, hw1, server(hw1))
	err := s.CheckIn(emp, factory, tzBerlin, hw2, server(hw2))

	if !errors.Is(err, attendance.ErrAlreadyCheckedIn) {
		t.Fatalf("expected ErrAlreadyCheckedIn, got %v", err)
	}
}

func TestCheckIn_AlreadyCheckedIn_StateUnchanged(t *testing.T) {
	// A second check-in while active must not mutate any state.
	s := attendance.NewStore()
	hw1 := fixed("2026-04-07T08:00:00Z")
	hw2 := fixed("2026-04-07T12:00:00Z")

	_ = s.CheckIn(emp, factory, tzBerlin, hw1, server(hw1))
	_ = s.CheckIn(emp, factory, tzBerlin, hw2, server(hw2)) // must be rejected

	// Check out using hw1's timeline — should still work cleanly.
	checkOut := fixed("2026-04-07T16:00:00Z") // 480 min from hw1
	shift, _, err := s.CheckOut(emp, checkOut, server(checkOut))

	if err != nil {
		t.Fatalf("checkout after rejected check-in failed: %v", err)
	}
	if shift.CheckInHW != hw1.UTC() {
		t.Errorf("expected original check-in time %v, got %v", hw1.UTC(), shift.CheckInHW)
	}
	if shift.Minutes != 480 {
		t.Errorf("expected 480 minutes from original check-in, got %d", shift.Minutes)
	}
}

func TestCheckIn_InvalidTimezone_ReturnsError(t *testing.T) {
	s := attendance.NewStore()
	hw := fixed("2026-04-07T08:00:00Z")

	err := s.CheckIn(emp, factory, "Not/ATimezone", hw, server(hw))

	if !errors.Is(err, attendance.ErrUnknownTimezone) {
		t.Fatalf("expected ErrUnknownTimezone, got %v", err)
	}
}

func TestCheckIn_ClockDrift_Rejected(t *testing.T) {
	s := attendance.NewStore()
	hw := fixed("2026-04-07T08:00:00Z")
	srv := hw.Add(10 * time.Minute) // 10 min > MaxClockDrift (5 min)

	err := s.CheckIn(emp, factory, tzBerlin, hw, srv)

	if !errors.Is(err, attendance.ErrClockDrift) {
		t.Fatalf("expected ErrClockDrift, got %v", err)
	}
}

// --- CheckOut ---

func TestCheckOut_Normal_ShiftMinutes(t *testing.T) {
	s := attendance.NewStore()
	checkIn := fixed("2026-04-07T08:00:00Z")
	checkOut := fixed("2026-04-07T16:30:00Z") // 8h30m = 510 minutes

	_ = s.CheckIn(emp, factory, tzBerlin, checkIn, server(checkIn))
	shift, _, err := s.CheckOut(emp, checkOut, server(checkOut))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shift.Minutes != 510 {
		t.Errorf("expected 510 minutes, got %d", shift.Minutes)
	}
}

func TestCheckOut_Normal_WeekStartIsMonday(t *testing.T) {
	s := attendance.NewStore()
	// 2026-04-07 is Tuesday in Europe/Berlin (CEST = UTC+2).
	// Local time: 2026-04-07T10:00:00+02:00
	// The Monday that starts this week is 2026-04-06 00:00:00 CEST = 2026-04-05T22:00:00Z.
	checkIn := fixed("2026-04-07T08:00:00Z") // 10:00 Berlin time
	checkOut := fixed("2026-04-07T14:00:00Z")

	_ = s.CheckIn(emp, factory, tzBerlin, checkIn, server(checkIn))
	shift, _, err := s.CheckOut(emp, checkOut, server(checkOut))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WeekStart should be 2026-04-06 00:00:00 CEST = 2026-04-05T22:00:00Z.
	wantWeekStart := fixed("2026-04-05T22:00:00Z")
	if !shift.WeekStart.Equal(wantWeekStart) {
		t.Errorf("expected WeekStart %v, got %v", wantWeekStart, shift.WeekStart)
	}
}

func TestCheckOut_ReturnsWeekTotalAtomically(t *testing.T) {
	s := attendance.NewStore()
	// Two shifts in the same Monday–Sunday week in Berlin time.
	ci1 := fixed("2026-04-07T08:00:00Z") // Tuesday W15-Berlin, 480 min
	co1 := fixed("2026-04-07T16:00:00Z")
	_ = s.CheckIn(emp, factory, tzBerlin, ci1, server(ci1))
	_, _, _ = s.CheckOut(emp, co1, server(co1))

	ci2 := fixed("2026-04-08T08:00:00Z") // Wednesday W15-Berlin, 480 min
	co2 := fixed("2026-04-08T16:00:00Z")
	_ = s.CheckIn(emp, factory, tzBerlin, ci2, server(ci2))
	_, weekTotal, err := s.CheckOut(emp, co2, server(co2))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if weekTotal != 960 {
		t.Errorf("expected week total 960, got %d", weekTotal)
	}
}

func TestCheckOut_NotCheckedIn(t *testing.T) {
	s := attendance.NewStore()
	hw := fixed("2026-04-07T16:00:00Z")

	_, _, err := s.CheckOut(emp, hw, server(hw))

	if !errors.Is(err, attendance.ErrNotCheckedIn) {
		t.Fatalf("expected ErrNotCheckedIn, got %v", err)
	}
}

func TestCheckOut_BeforeCheckIn(t *testing.T) {
	s := attendance.NewStore()
	checkIn := fixed("2026-04-07T08:00:00Z")
	checkOut := fixed("2026-04-07T07:59:00Z") // one minute before check-in

	_ = s.CheckIn(emp, factory, tzBerlin, checkIn, server(checkIn))
	_, _, err := s.CheckOut(emp, checkOut, server(checkOut))

	if !errors.Is(err, attendance.ErrCheckOutBeforeCheckIn) {
		t.Fatalf("expected ErrCheckOutBeforeCheckIn, got %v", err)
	}
}

func TestCheckOut_EqualToCheckIn(t *testing.T) {
	s := attendance.NewStore()
	hw := fixed("2026-04-07T08:00:00Z")

	_ = s.CheckIn(emp, factory, tzBerlin, hw, server(hw))
	_, _, err := s.CheckOut(emp, hw, server(hw)) // equal, not strictly after

	if !errors.Is(err, attendance.ErrCheckOutBeforeCheckIn) {
		t.Fatalf("expected ErrCheckOutBeforeCheckIn, got %v", err)
	}
}

func TestCheckOut_ClockDrift_Rejected(t *testing.T) {
	s := attendance.NewStore()
	checkIn := fixed("2026-04-07T08:00:00Z")
	_ = s.CheckIn(emp, factory, tzBerlin, checkIn, server(checkIn))

	checkOut := fixed("2026-04-07T16:00:00Z")
	srv := checkOut.Add(10 * time.Minute) // 10 min drift > MaxClockDrift

	_, _, err := s.CheckOut(emp, checkOut, srv)

	if !errors.Is(err, attendance.ErrClockDrift) {
		t.Fatalf("expected ErrClockDrift, got %v", err)
	}
}

// --- WeeklyMinutes ---

func TestWeeklyMinutes_SumsCurrentWeekOnly(t *testing.T) {
	s := attendance.NewStore()

	// Week of 2026-04-06 (Mon) in Berlin: 480 min.
	ci1 := fixed("2026-04-07T08:00:00Z") // Tuesday
	co1 := fixed("2026-04-07T16:00:00Z")
	_ = s.CheckIn(emp, factory, tzBerlin, ci1, server(ci1))
	_, _, _ = s.CheckOut(emp, co1, server(co1))

	// Next week (2026-04-13 Mon): 480 min — must NOT appear in previous week's total.
	ci2 := fixed("2026-04-14T08:00:00Z") // Tuesday next week
	co2 := fixed("2026-04-14T16:00:00Z")
	_ = s.CheckIn(emp, factory, tzBerlin, ci2, server(ci2))
	_, _, _ = s.CheckOut(emp, co2, server(co2))

	total, err := s.WeeklyMinutes(emp, ci1, tzBerlin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 480 {
		t.Errorf("expected 480 (current week only), got %d", total)
	}
}

func TestWeeklyMinutes_WeekBoundary_SundayVsMonday_LocalTime(t *testing.T) {
	s := attendance.NewStore()

	// Key scenario: check whether a shift that is Sunday in local time
	// (but Monday in UTC) is placed in the PREVIOUS week.
	//
	// America/Sao_Paulo is UTC-3.
	// 2026-04-13T01:00:00Z = 2026-04-12T22:00:00-03:00 → Sunday locally → previous week.
	// 2026-04-13T10:00:00Z = 2026-04-13T07:00:00-03:00 → Monday locally → new week.
	const tzSP = "America/Sao_Paulo"

	// Sunday-in-local shift: 60 min (01:00–02:00 UTC = 22:00–23:00 local Sunday).
	ciSun := fixed("2026-04-13T01:00:00Z")
	coSun := fixed("2026-04-13T02:00:00Z")
	_ = s.CheckIn(emp, factory, tzSP, ciSun, server(ciSun))
	_, _, _ = s.CheckOut(emp, coSun, server(coSun))

	// Monday-in-local shift: 480 min (10:00–18:00 UTC = 07:00–15:00 local Monday).
	ciMon := fixed("2026-04-13T10:00:00Z")
	coMon := fixed("2026-04-13T18:00:00Z")
	_ = s.CheckIn(emp, factory, tzSP, ciMon, server(ciMon))
	_, _, _ = s.CheckOut(emp, coMon, server(coMon))

	sundayWeekTotal, err := s.WeeklyMinutes(emp, ciSun, tzSP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mondayWeekTotal, err := s.WeeklyMinutes(emp, ciMon, tzSP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// If the implementation mistakenly used UTC, both shifts would appear on
	// Monday (W16) and the Sunday week total would be 0 — wrong.
	if sundayWeekTotal != 60 {
		t.Errorf("Sunday local shift: expected 60 min in its week, got %d", sundayWeekTotal)
	}
	if mondayWeekTotal != 480 {
		t.Errorf("Monday local shift: expected 480 min in its week, got %d", mondayWeekTotal)
	}
}

func TestWeeklyMinutes_NoShifts(t *testing.T) {
	s := attendance.NewStore()
	ref := fixed("2026-04-07T00:00:00Z")

	total, err := s.WeeklyMinutes("unknown-employee", ref, tzBerlin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 for unknown employee, got %d", total)
	}
}

func TestWeeklyMinutes_InvalidTimezone(t *testing.T) {
	s := attendance.NewStore()
	ref := fixed("2026-04-07T00:00:00Z")

	_, err := s.WeeklyMinutes(emp, ref, "Bad/Zone")
	if !errors.Is(err, attendance.ErrUnknownTimezone) {
		t.Fatalf("expected ErrUnknownTimezone, got %v", err)
	}
}
