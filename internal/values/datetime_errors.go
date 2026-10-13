package values

import (
	"fmt"
	"time"
)

// ErrDatetimeFormat describes an unsupported datetime format.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrDatetimeFormat struct {
	Format string
}

// NewErrDatetimeFormat constructs an error for an unsupported datetime format.
func NewErrDatetimeFormat(format string) *ErrDatetimeFormat {
	return &ErrDatetimeFormat{
		Format: format,
	}
}

// Error describes the unsupported format.
func (e *ErrDatetimeFormat) Error() string {
	return "unknown time format " + e.Format
}

// ErrDatetimeConstraint describes a datetime outside a strict time bound.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrDatetimeConstraint struct {
	Value    time.Time
	Bound    time.Time
	Relation string
}

// NewErrDatetimeConstraint constructs an error for a failed time bound.
func NewErrDatetimeConstraint(value, bound time.Time, relation string) *ErrDatetimeConstraint {
	return &ErrDatetimeConstraint{
		Value:    value,
		Bound:    bound,
		Relation: relation,
	}
}

// Error describes the required ordering of the datetime and its bound.
func (e *ErrDatetimeConstraint) Error() string {
	return fmt.Sprintf("%s must happen %s %s", e.Value, e.Relation, e.Bound)
}
