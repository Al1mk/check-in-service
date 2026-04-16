package httpapi

// EventRequest is the JSON body accepted by POST /events.
type EventRequest struct {
	EmployeeID        string `json:"employee_id"`
	FactoryID         string `json:"factory_id"`
	FactoryLocation   string `json:"factory_location"`  // IANA timezone, e.g. "Europe/Berlin"
	HardwareTimestamp string `json:"hardware_timestamp"` // RFC3339
	EventType         string `json:"event_type"`         // "check_in" or "check_out"
}

// CheckOutResponse is the JSON body returned on a successful check-out.
type CheckOutResponse struct {
	EmployeeID   string `json:"employee_id"`
	ShiftMinutes int    `json:"shift_minutes"`
	WeekMinutes  int    `json:"week_minutes"`
	WeekKey      string `json:"week"`
}
