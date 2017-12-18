package nrpe

// Result is the nrpe result type
type Result int

// nrpe result values
const (
	OK       = 0
	WARNING  = 1
	CRITICAL = 2
	UNKNOWN  = 3
)
