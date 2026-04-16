package httpapi

import "errors"

// EventRequest is the JSON body accepted by POST /events.
type EventRequest struct {
	EmployeeID        string `json:"employee_id"`
	FactoryID         string `json:"factory_id"`
	FactoryLocation   string `json:"factory_location"`   // IANA timezone, e.g. "Europe/Berlin"
	HardwareTimestamp string `json:"hardware_timestamp"` // RFC3339
	EventType         string `json:"event_type"`         // "check_in" or "check_out"
}

// validate checks that all required fields are present and event_type is known.
//
// Note on factory_location for check-out: the field is accepted and validated
// as non-empty, but the store uses the timezone recorded at check-in for shift
// closure and week calculation. The check-out value is not forwarded to the store.
func (r EventRequest) validate() error {
	switch {
	case r.EmployeeID == "":
		return errors.New("employee_id is required")
	case r.FactoryID == "":
		return errors.New("factory_id is required")
	case r.FactoryLocation == "":
		return errors.New("factory_location is required")
	case r.HardwareTimestamp == "":
		return errors.New("hardware_timestamp is required")
	case r.EventType != "check_in" && r.EventType != "check_out":
		return errors.New("event_type must be \"check_in\" or \"check_out\"")
	}
	return nil
}

// CheckOutResponse is the JSON body returned on a successful check-out.
type CheckOutResponse struct {
	EmployeeID   string `json:"employee_id"`
	ShiftMinutes int    `json:"shift_minutes"`
	WeekMinutes  int    `json:"week_minutes"`
}
