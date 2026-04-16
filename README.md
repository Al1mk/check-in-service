# check-in-service

A Go HTTP service that records employee check-in and check-out events from factory card readers, tracks active shifts in memory, calculates worked minutes on check-out, and forwards completed shift data asynchronously to a mock external system.

**Status:** implementation in progress.

---

## Package structure

```text
internal/
  httpapi/      HTTP handlers and JSON payload types (transport boundary)
  attendance/   Domain state: open shifts and shift history (business logic)
  forwarding/   Background worker that forwards shift data to the external system
  mock/         Fake external system handler used for local testing
