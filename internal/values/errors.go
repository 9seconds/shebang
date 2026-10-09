package values

import (
	"fmt"
)

// Number includes the numeric representations supported by value validators.
type Number interface {
	~float64 | ~int64
}

// NegativeLengthError describes a negative string-length constraint.
type NegativeLengthError struct {
	Length int64
}

// NewNegativeLengthError constructs an error for a negative length bound.
func NewNegativeLengthError(length int64) *NegativeLengthError {
	return &NegativeLengthError{Length: length}
}

// Error describes the rejected length bound.
func (e *NegativeLengthError) Error() string {
	return fmt.Sprintf("length must be positive, not %d", e.Length)
}

// LengthConstraintKind identifies the string-length constraint that failed.
type LengthConstraintKind int

const (
	// LengthConstraintMinimum indicates a value shorter than the minimum.
	LengthConstraintMinimum LengthConstraintKind = iota
	// LengthConstraintMaximum indicates a value longer than the maximum.
	LengthConstraintMaximum
)

// LengthConstraintError describes a violation measured in Unicode code points.
type LengthConstraintError struct {
	Kind     LengthConstraintKind
	Expected int
	Actual   int
}

// NewLengthConstraintError constructs an error for a string-length violation.
func NewLengthConstraintError(
	kind LengthConstraintKind,
	expected, actual int,
) *LengthConstraintError {
	return &LengthConstraintError{
		Kind:     kind,
		Expected: expected,
		Actual:   actual,
	}
}

// Error describes the failed length constraint.
func (e *LengthConstraintError) Error() string {
	switch e.Kind {
	case LengthConstraintMinimum:
		return fmt.Sprintf("minimum length must be at least %d, got %d", e.Expected, e.Actual)
	case LengthConstraintMaximum:
		return fmt.Sprintf("minimum length must be at most %d, got %d", e.Expected, e.Actual)
	}

	return fmt.Sprintf("expected length %d, got %d", e.Expected, e.Actual)
}

// RegexMismatchError describes a string that does not match a regular expression.
type RegexMismatchError struct {
	Value   string
	Pattern string
}

// NewRegexMismatchError constructs an error for a regular-expression mismatch.
func NewRegexMismatchError(value, pattern string) *RegexMismatchError {
	return &RegexMismatchError{Value: value, Pattern: pattern}
}

// Error describes the rejected string and regular expression.
func (e *RegexMismatchError) Error() string {
	return fmt.Sprintf("%s does not match %s", e.Value, e.Pattern)
}

// NumConstraintKind identifies the inclusive numeric bound that failed.
type NumConstraintKind int

const (
	// NumConstraintMin indicates a value below its minimum.
	NumConstraintMin NumConstraintKind = iota
	// NumConstraintMax indicates a value above its maximum.
	NumConstraintMax
)

// NumConstraintError describes a violation of an inclusive numeric bound.
type NumConstraintError[T Number] struct {
	Kind     NumConstraintKind
	Expected T
	Actual   T
}

// NewNumConstraintError constructs an error for a numeric bound violation.
func NewNumConstraintError[T Number](
	kind NumConstraintKind,
	expected, actual T,
) *NumConstraintError[T] {
	return &NumConstraintError[T]{
		Kind:     kind,
		Expected: expected,
		Actual:   actual,
	}
}

// Error describes the failed numeric bound and the supplied value.
func (e *NumConstraintError[T]) Error() string {
	switch e.Kind {
	case NumConstraintMin:
		return fmt.Sprintf("number must be at least %v, got %v", e.Expected, e.Actual)
	case NumConstraintMax:
		return fmt.Sprintf("number must be at most %v, got %v", e.Expected, e.Actual)
	}

	return fmt.Sprintf("expected %v, got %v", e.Expected, e.Actual)
}
