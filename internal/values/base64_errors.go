package values

import (
	"errors"
	"fmt"
)

// ErrBase64NegativeLength indicates a negative decoded-byte length bound.
var ErrBase64NegativeLength = errors.New("length cannot be negative")

// ErrBase64Format describes an unsupported Base64 format.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrBase64Format struct {
	Format string
}

// NewErrBase64Format constructs an error for an unsupported Base64 format.
func NewErrBase64Format(format string) *ErrBase64Format {
	return &ErrBase64Format{
		Format: format,
	}
}

// Error describes the unsupported format.
func (e *ErrBase64Format) Error() string {
	return "unknown format " + e.Format
}

// ErrBase64LengthConstraint describes a decoded-byte length outside a bound.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrBase64LengthConstraint struct {
	Relation string
	Expected int64
	Actual   int64
}

// NewErrBase64LengthConstraint constructs an error for a decoded-byte length violation.
func NewErrBase64LengthConstraint(
	relation string,
	expected, actual int64,
) *ErrBase64LengthConstraint {
	return &ErrBase64LengthConstraint{
		Relation: relation,
		Expected: expected,
		Actual:   actual,
	}
}

// Error describes the required decoded-byte length.
func (e *ErrBase64LengthConstraint) Error() string {
	return fmt.Sprintf("length must be %s %d bytes", e.Relation, e.Expected)
}
