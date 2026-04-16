package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Al1mk/check-in-service/internal/attendance"
	"github.com/Al1mk/check-in-service/internal/httpapi"
)

// newHandler returns a handler wired to a fresh store.
func newHandler() *httpapi.EventHandler {
	return httpapi.NewEventHandler(attendance.NewStore())
}

// post sends a POST /events request with the given body and returns the recorder.
func post(t *testing.T, h http.Handler, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// hwNow returns an RFC3339 timestamp 10 seconds in the past so the drift guard
// passes when the handler calls time.Now() at receipt.
func hwNow() string {
	return time.Now().Add(-10 * time.Second).UTC().Format(time.RFC3339)
}

// basePayload returns a fully valid check-in payload.
func basePayload() map[string]string {
	return map[string]string{
		"employee_id":        "E001",
		"factory_id":         "F-Munich-01",
		"factory_location":   "Europe/Berlin",
		"hardware_timestamp": hwNow(),
		"event_type":         "check_in",
	}
}

// --- request validation ---

func TestPost_MalformedJSON_Returns400(t *testing.T) {
	h := newHandler()
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPost_MissingEmployeeID_Returns422(t *testing.T) {
	h := newHandler()
	p := basePayload()
	delete(p, "employee_id")
	w := post(t, h, p)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

func TestPost_MissingFactoryID_Returns422(t *testing.T) {
	h := newHandler()
	p := basePayload()
	delete(p, "factory_id")
	w := post(t, h, p)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

func TestPost_MissingFactoryLocation_Returns422(t *testing.T) {
	h := newHandler()
	p := basePayload()
	delete(p, "factory_location")
	w := post(t, h, p)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

func TestPost_MissingHardwareTimestamp_Returns422(t *testing.T) {
	h := newHandler()
	p := basePayload()
	delete(p, "hardware_timestamp")
	w := post(t, h, p)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

func TestPost_InvalidEventType_Returns422(t *testing.T) {
	h := newHandler()
	p := basePayload()
	p["event_type"] = "arrive"
	w := post(t, h, p)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

func TestPost_MalformedTimestamp_Returns422(t *testing.T) {
	h := newHandler()
	p := basePayload()
	p["hardware_timestamp"] = "not-a-date"
	w := post(t, h, p)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

func TestPost_InvalidTimezone_Returns422(t *testing.T) {
	h := newHandler()
	p := basePayload()
	p["factory_location"] = "Not/ATimezone"
	w := post(t, h, p)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// --- check-in success ---

func TestPost_CheckIn_Returns204(t *testing.T) {
	h := newHandler()
	w := post(t, h, basePayload())
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestPost_CheckIn_Returns204_EmptyBody(t *testing.T) {
	h := newHandler()
	w := post(t, h, basePayload())
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body on 204, got %q", w.Body.String())
	}
}

// --- check-in conflict ---

func TestPost_CheckIn_AlreadyCheckedIn_Returns409(t *testing.T) {
	h := newHandler()
	p := basePayload()
	post(t, h, p)          // first succeeds
	w := post(t, h, p)     // second is a conflict
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

// --- check-out success ---

func TestPost_CheckOut_Returns200WithBody(t *testing.T) {
	h := newHandler()

	// Both timestamps must be within MaxClockDrift (5 min) of time.Now() at receipt.
	// check-out is strictly after check-in.
	checkInHW := time.Now().Add(-30 * time.Second).UTC()
	checkOutHW := time.Now().Add(-10 * time.Second).UTC()

	post(t, h, map[string]string{
		"employee_id":        "E001",
		"factory_id":         "F-Munich-01",
		"factory_location":   "Europe/Berlin",
		"hardware_timestamp": checkInHW.Format(time.RFC3339),
		"event_type":         "check_in",
	})

	w := post(t, h, map[string]string{
		"employee_id":        "E001",
		"factory_id":         "F-Munich-01",
		"factory_location":   "Europe/Berlin",
		"hardware_timestamp": checkOutHW.Format(time.RFC3339),
		"event_type":         "check_out",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp httpapi.CheckOutResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.EmployeeID != "E001" {
		t.Errorf("employee_id: want E001, got %q", resp.EmployeeID)
	}
	if resp.ShiftMinutes < 0 {
		t.Errorf("shift_minutes must be >= 0, got %d", resp.ShiftMinutes)
	}
	if resp.WeekMinutes < 0 {
		t.Errorf("week_minutes must be >= 0, got %d", resp.WeekMinutes)
	}
}

// --- check-out domain errors ---

func TestPost_CheckOut_NotCheckedIn_Returns409(t *testing.T) {
	h := newHandler()
	p := basePayload()
	p["event_type"] = "check_out"
	w := post(t, h, p)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

// --- error response shape ---

func TestPost_ErrorResponse_IsJSON(t *testing.T) {
	h := newHandler()
	r := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString("bad"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Errorf("error body is not valid JSON: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Error("error body missing \"error\" key")
	}
}

func TestPost_CheckIn_204_NoContentType(t *testing.T) {
	h := newHandler()
	w := post(t, h, basePayload())
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204")
	}
	if ct := w.Header().Get("Content-Type"); ct == "application/json" {
		t.Error("204 response must not have application/json Content-Type")
	}
}
