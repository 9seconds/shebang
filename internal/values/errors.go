package values

import "fmt"

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

// IntConstraintKind identifies the integer bound that failed.
type IntConstraintKind int

const (
	// IntConstraintMin indicates an integer below its inclusive minimum.
	IntConstraintMin IntConstraintKind = iota
	// IntConstraintMax indicates an integer above its inclusive maximum.
	IntConstraintMax
)

// IntConstraintError describes a violation of an inclusive integer bound.
type IntConstraintError struct {
	Kind     IntConstraintKind
	Expected int64
	Actual   int64
}

// NewIntConstraintError constructs an error for an integer bound violation.
func NewIntConstraintError(kind IntConstraintKind, expected, actual int64) *IntConstraintError {
	return &IntConstraintError{
		Kind:     kind,
		Expected: expected,
		Actual:   actual,
	}
}

// Error describes the failed integer bound and the supplied value.
func (e *IntConstraintError) Error() string {
	switch e.Kind {
	case IntConstraintMin:
		return fmt.Sprintf("number must be at least %d, got %d", e.Expected, e.Actual)
	case IntConstraintMax:
		return fmt.Sprintf("number must be at most %d, got %d", e.Expected, e.Actual)
	}

	return fmt.Sprintf("expected %d, got %d", e.Expected, e.Actual)
}
