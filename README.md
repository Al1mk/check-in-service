# check-in-service

A Go HTTP service that records employee check-in and check-out events from factory card readers, tracks active shifts in memory, calculates worked minutes on check-out, and forwards completed shift data asynchronously to an external system.

**Status:** Phase 4 complete — forwarding worker and mock endpoint implemented.

---

## Package structure

```text
internal/
  httpapi/      HTTP handlers and JSON payload types (transport boundary)
  attendance/   Domain state: open shifts and shift history (business logic)
  forwarding/   Background worker that forwards completed shifts to the external system
  mock/         Fake external system that randomly fails or delays to exercise retries
```

---

## API

### `POST /events`

Records a check-in or check-out event from a factory card reader.

**Request body** (JSON):

| Field | Type | Required | Description |
|---|---|---|---|
| `employee_id` | string | yes | Unique identifier for the employee |
| `factory_id` | string | yes | Identifier for the factory |
| `factory_location` | string | yes | IANA timezone, e.g. `"Europe/Berlin"` |
| `hardware_timestamp` | string | yes | RFC3339 timestamp from the card reader |
| `event_type` | string | yes | `"check_in"` or `"check_out"` |

**Check-in success** — `204 No Content`, empty body.

**Check-out success** — `200 OK`:

```json
{
  "employee_id": "E001",
  "shift_minutes": 480,
  "week_minutes": 960
}
```

`shift_minutes` — duration of the just-closed shift in whole minutes.  
`week_minutes` — running total for the Monday–Sunday calendar week (in factory-local time) that contains the shift.

**Error responses** — all errors return `Content-Type: application/json` with an `"error"` key:

```json
{ "error": "description" }
```

| Status | Condition |
|---|---|
| `400 Bad Request` | Body is not valid JSON |
| `409 Conflict` | `check_in` while already checked in; `check_out` while not checked in |
| `422 Unprocessable Entity` | Missing required field; invalid `event_type`; malformed `hardware_timestamp`; unknown `factory_location` timezone; hardware timestamp exceeds clock-drift limit (5 min) |

---

## Running

```bash
go run .        # starts on :8080
go test ./...   # all unit tests, no external dependencies
```

---

## Design notes

**Clock drift guard** — the service compares the card-reader hardware timestamp against `time.Now()` at receipt. Deviations beyond `MaxClockDrift` (5 minutes) are rejected with 422. A spike in drift rejections indicates a card reader with a broken clock and should trigger an operational alert.

**Week boundary** — the Monday–Sunday week is computed in factory-local time (`factory_location`), not UTC. A shift at 23:00 on a Sunday in Berlin belongs to the previous week, even if it falls on a Monday in UTC.

**Missed check-out** — a second `check_in` while an active shift exists is rejected with 409. No automatic repair is performed; resolution requires an administrative process outside this service.

**Forwarding** — on successful check-out the handler enqueues a `forwarding.Job` onto a buffered channel (capacity 100) using a non-blocking `select`. The 200 response confirms the shift was processed locally in the attendance store; it does not guarantee forwarding delivery. If the internal channel is full the job is logged and dropped — forwarding is best-effort in this single-process design. A single background `RunWorker` goroutine reads from the channel and POSTs to `POST /mock/recording` with a 5-second outbound timeout. On failure it retries up to 3 times with delays of 1 s, 2 s, and 4 s before logging and discarding the job.

**Forwarding durability** — this implementation intentionally keeps forwarding best-effort. The HTTP response reflects committed local state; asynchronous delivery to the external system may be lost if the process restarts or the queue fills. A production design would close this gap with a durable queue or a transactional outbox pattern so that no committed shift is ever silently dropped.

**Mock endpoint** — `POST /mock/recording` returns 500 ~30 % of the time, delays 2–5 s before 200 ~20 % of the time, and returns 200 immediately ~50 % of the time. This exercises all retry paths without external infrastructure.
