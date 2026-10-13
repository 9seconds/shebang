package values

import "fmt"

// ErrPortRange describes a port with an unexpected range classification.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrPortRange struct {
	Port     uint16
	Range    string
	Expected bool
}

// NewErrPortRange constructs an error for a failed port range constraint.
func NewErrPortRange(port uint16, category string, expected bool) *ErrPortRange {
	return &ErrPortRange{
		Port:     port,
		Range:    category,
		Expected: expected,
	}
}

// Error describes whether the port must belong to the configured range.
func (e *ErrPortRange) Error() string {
	if e.Expected {
		return fmt.Sprintf("port %d is not %s", e.Port, e.Range)
	}

	return fmt.Sprintf("port %d is %s", e.Port, e.Range)
}
