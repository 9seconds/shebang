package values

import (
	"fmt"
	"time"
)

// DatetimeFormatError describes an unsupported datetime format.
type DatetimeFormatError struct {
	Format string
}

// NewDatetimeFormatError constructs an error for an unsupported datetime format.
func NewDatetimeFormatError(format string) *DatetimeFormatError {
	return &DatetimeFormatError{
		Format: format,
	}
}

// Error describes the unsupported format.
func (e *DatetimeFormatError) Error() string {
	return "unknown time format " + e.Format
}

// DatetimeConstraintError describes a datetime outside a strict time bound.
type DatetimeConstraintError struct {
	Value    time.Time
	Bound    time.Time
	Relation string
}

// NewDatetimeConstraintError constructs an error for a failed time bound.
func NewDatetimeConstraintError(value, bound time.Time, relation string) *DatetimeConstraintError {
	return &DatetimeConstraintError{
		Value:    value,
		Bound:    bound,
		Relation: relation,
	}
}

// Error describes the required ordering of the datetime and its bound.
func (e *DatetimeConstraintError) Error() string {
	return fmt.Sprintf("%s must happen %s %s", e.Value, e.Relation, e.Bound)
}
