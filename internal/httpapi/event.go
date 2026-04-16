// Package httpapi contains the HTTP handlers and transport types for the check-in service.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Al1mk/check-in-service/internal/attendance"
)

// EventHandler handles POST /events.
type EventHandler struct {
	store *attendance.Store
}

// NewEventHandler constructs an EventHandler wired to the given store.
func NewEventHandler(store *attendance.Store) *EventHandler {
	return &EventHandler{store: store}
}

// ServeHTTP validates the incoming event and updates attendance state.
// On check-out it returns shift minutes and the current calendar-week total.
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := req.validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	hwTime, err := time.Parse(time.RFC3339, req.HardwareTimestamp)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "hardware_timestamp must be RFC3339")
		return
	}

	serverTime := time.Now()

	switch req.EventType {
	case "check_in":
		h.handleCheckIn(w, req, hwTime, serverTime)
	case "check_out":
		h.handleCheckOut(w, req, hwTime, serverTime)
	}
}

func (h *EventHandler) handleCheckIn(w http.ResponseWriter, req EventRequest, hwTime, serverTime time.Time) {
	if err := h.store.CheckIn(req.EmployeeID, req.FactoryID, req.FactoryLocation, hwTime, serverTime); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EventHandler) handleCheckOut(w http.ResponseWriter, req EventRequest, hwTime, serverTime time.Time) {
	shift, weekMinutes, err := h.store.CheckOut(req.EmployeeID, hwTime, serverTime)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	resp := CheckOutResponse{
		EmployeeID:   req.EmployeeID,
		ShiftMinutes: shift.Minutes,
		WeekMinutes:  weekMinutes,
	}
	writeJSON(w, http.StatusOK, resp)
}

// writeStoreError maps known attendance sentinel errors to HTTP status codes.
//
// Conflict errors (state precondition not met):
//   - ErrAlreadyCheckedIn  → 409: employee has an open shift; cannot open another
//   - ErrNotCheckedIn      → 409: no open shift to close
//
// Unprocessable errors (request data is invalid):
//   - ErrCheckOutBeforeCheckIn → 422: checkout timestamp not after check-in
//   - ErrClockDrift            → 422: hardware timestamp too far from server time
//   - ErrUnknownTimezone       → 422: factory_location is not a valid IANA name
func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, attendance.ErrAlreadyCheckedIn):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, attendance.ErrNotCheckedIn):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, attendance.ErrCheckOutBeforeCheckIn):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, attendance.ErrClockDrift):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, attendance.ErrUnknownTimezone):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// writeJSON encodes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error body: {"error": "message"}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
