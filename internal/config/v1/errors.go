package v1

import "fmt"

// ShortNamePatternError describes a shorthand that fails its required pattern.
type ShortNamePatternError struct {
	Pattern string
}

// NewShortNamePatternError constructs an error for an invalid shorthand pattern.
func NewShortNamePatternError(pattern string) *ShortNamePatternError {
	return &ShortNamePatternError{Pattern: pattern}
}

// Error describes the required shorthand pattern.
func (e *ShortNamePatternError) Error() string {
	return fmt.Sprintf("must comply %s regexp", e.Pattern)
}

// NamePatternError describes a declaration name that fails its required pattern.
type NamePatternError struct {
	Value   string
	Pattern string
}

// NewNamePatternError constructs an error for an invalid declaration name.
func NewNamePatternError(value, pattern string) *NamePatternError {
	return &NamePatternError{Value: value, Pattern: pattern}
}

// Error describes the rejected name and its required pattern.
func (e *NamePatternError) Error() string {
	return fmt.Sprintf("value %s does not match regex %s", e.Value, e.Pattern)
}

// ShortNameLengthError describes a shorthand containing other than one rune.
type ShortNameLengthError struct {
	Count int
}

// NewShortNameLengthError constructs an error for an invalid shorthand length.
func NewShortNameLengthError(count int) *ShortNameLengthError {
	return &ShortNameLengthError{
		Count: count,
	}
}

// Error describes the required shorthand length and the actual rune count.
func (e *ShortNameLengthError) Error() string {
	return fmt.Sprintf("short must contain 1 character, not %d", e.Count)
}

// VarArgLimitError describes a vararg bound exceeding MaxVarArgs.
type VarArgLimitError struct {
	Bound string
	Count int64
}

// NewVarArgLimitError constructs an error for an excessive vararg bound.
func NewVarArgLimitError(bound string, count int64) *VarArgLimitError {
	return &VarArgLimitError{
		Bound: bound,
		Count: count,
	}
}

// Error describes the limit on explicitly configured vararg bounds.
func (e *VarArgLimitError) Error() string {
	return fmt.Sprintf("if you use more than %d max arguments, do not limit them", MaxVarArgs)
}

// VarArgBoundsError describes a minimum exceeding a finite maximum.
type VarArgBoundsError struct {
	Min int64
	Max int64
}

// NewVarArgBoundsError constructs an error for reversed vararg bounds.
func NewVarArgBoundsError(minimum, maximum int64) *VarArgBoundsError {
	return &VarArgBoundsError{
		Min: minimum,
		Max: maximum,
	}
}

// Error describes the conflicting minimum and maximum counts.
func (e *VarArgBoundsError) Error() string {
	return fmt.Sprintf("min-count %d is greater than max-count %d", e.Min, e.Max)
}

// ArgumentCountKind identifies the argument count requirement that failed.
type ArgumentCountKind int

const (
	// ArgumentCountMinimum indicates too few total positional arguments.
	ArgumentCountMinimum ArgumentCountKind = iota
	// ArgumentCountUnexpectedVarArgs indicates extra arguments without a vararg declaration.
	ArgumentCountUnexpectedVarArgs
	// ArgumentCountMaximumVarArgs indicates too many items in a variadic group.
	ArgumentCountMaximumVarArgs
)

// ArgumentCountError describes a positional or variadic argument count violation.
// Expected and Actual count total arguments for ArgumentCountMinimum and variadic
// items for the other kinds.
type ArgumentCountError struct {
	Expected int
	Actual   int
	Kind     ArgumentCountKind
}

// NewArgumentCountError constructs an error for the failed count requirement.
func NewArgumentCountError(kind ArgumentCountKind, expected, actual int) *ArgumentCountError {
	return &ArgumentCountError{
		Expected: expected,
		Actual:   actual,
		Kind:     kind,
	}
}

// Error describes the failed count requirement and the supplied argument count.
func (e *ArgumentCountError) Error() string {
	switch e.Kind {
	case ArgumentCountMinimum:
		return fmt.Sprintf("there must be at least %d arguments, got %d", e.Expected, e.Actual)
	case ArgumentCountUnexpectedVarArgs:
		return fmt.Sprintf("expected %d variadic arguments, got %d", e.Expected, e.Actual)
	case ArgumentCountMaximumVarArgs:
		return fmt.Sprintf("there must be at most %d variadic arguments, got %d", e.Expected, e.Actual)
	}

	return fmt.Sprintf("expected %d arguments, got %d", e.Expected, e.Actual)
}
