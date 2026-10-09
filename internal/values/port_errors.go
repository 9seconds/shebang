package values

import "fmt"

// PortRangeError describes a port with an unexpected range classification.
type PortRangeError struct {
	Port     uint16
	Range    string
	Expected bool
}

// NewPortRangeError constructs an error for a failed port range constraint.
func NewPortRangeError(port uint16, category string, expected bool) *PortRangeError {
	return &PortRangeError{
		Port:     port,
		Range:    category,
		Expected: expected,
	}
}

// Error describes whether the port must belong to the configured range.
func (e *PortRangeError) Error() string {
	if e.Expected {
		return fmt.Sprintf("port %d is not %s", e.Port, e.Range)
	}

	return fmt.Sprintf("port %d is %s", e.Port, e.Range)
}
