package values

import (
	"errors"
	"fmt"
)

// ErrStrNotPrintable indicates a string containing a non-printable character.
var ErrStrNotPrintable = errors.New("is not printable")

// ErrStrNegativeLength describes a negative string-length constraint.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrStrNegativeLength struct {
	Length int64
}

// NewErrStrNegativeLength constructs an error for a negative length bound.
func NewErrStrNegativeLength(length int64) *ErrStrNegativeLength {
	return &ErrStrNegativeLength{
		Length: length,
	}
}

// Error describes the rejected length bound.
func (e *ErrStrNegativeLength) Error() string {
	return fmt.Sprintf("length must be non-negative, not %d", e.Length)
}

// StrLengthConstraintKind identifies the string-length constraint that failed.
type StrLengthConstraintKind int

const (
	// StrLengthConstraintMinimum indicates a value shorter than the minimum.
	StrLengthConstraintMinimum StrLengthConstraintKind = iota
	// StrLengthConstraintMaximum indicates a value longer than the maximum.
	StrLengthConstraintMaximum
)

// ErrStrLengthConstraint describes a violation measured in Unicode code points.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrStrLengthConstraint struct {
	Kind     StrLengthConstraintKind
	Expected int
	Actual   int
}

// NewErrStrLengthConstraint constructs an error for a string-length violation.
func NewErrStrLengthConstraint(
	kind StrLengthConstraintKind,
	expected, actual int,
) *ErrStrLengthConstraint {
	return &ErrStrLengthConstraint{
		Kind:     kind,
		Expected: expected,
		Actual:   actual,
	}
}

// Error describes the failed length constraint.
func (e *ErrStrLengthConstraint) Error() string {
	switch e.Kind {
	case StrLengthConstraintMinimum:
		return fmt.Sprintf("minimum length must be at least %d, got %d", e.Expected, e.Actual)
	case StrLengthConstraintMaximum:
		return fmt.Sprintf("maximum length must be at most %d, got %d", e.Expected, e.Actual)
	}

	return fmt.Sprintf("expected length %d, got %d", e.Expected, e.Actual)
}

// ErrStrRegexMismatch describes a string that does not match a regular expression.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrStrRegexMismatch struct {
	Value   string
	Pattern string
}

// NewErrStrRegexMismatch constructs an error for a regular-expression mismatch.
func NewErrStrRegexMismatch(value, pattern string) *ErrStrRegexMismatch {
	return &ErrStrRegexMismatch{
		Value:   value,
		Pattern: pattern,
	}
}

// Error describes the rejected string and regular expression.
func (e *ErrStrRegexMismatch) Error() string {
	return fmt.Sprintf("%s does not match %s", e.Value, e.Pattern)
}
