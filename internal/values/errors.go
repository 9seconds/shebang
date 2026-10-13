package values

import (
	"errors"
	"fmt"
)

var (
	// ErrUnknownValueType indicates an unsupported value type.
	ErrUnknownValueType = errors.New("unknown type")
	// ErrUnknownSubvalidator indicates an unsupported subvalidator.
	ErrUnknownSubvalidator = errors.New("unknown subvalidator")
	// ErrUnknownProperty indicates an unsupported value property.
	ErrUnknownProperty = errors.New("unknown property")
	// ErrNoProperties indicates that the value does not support properties.
	ErrNoProperties = errors.New("does not support properties")
)

// Number includes the numeric representations supported by value validators.
type Number interface {
	~float64 | ~int64
}

// NumConstraintKind identifies the inclusive numeric bound that failed.
type NumConstraintKind int

const (
	// NumConstraintMin indicates a value below its minimum.
	NumConstraintMin NumConstraintKind = iota
	// NumConstraintMax indicates a value above its maximum.
	NumConstraintMax
)

// ErrNumConstraint describes a violation of an inclusive numeric bound.
//
//nolint:errname // Shared errors follow the project's Err-prefixed naming convention.
type ErrNumConstraint[T Number] struct {
	Kind     NumConstraintKind
	Expected T
	Actual   T
}

// NewErrNumConstraint constructs an error for a numeric bound violation.
func NewErrNumConstraint[T Number](
	kind NumConstraintKind,
	expected, actual T,
) *ErrNumConstraint[T] {
	return &ErrNumConstraint[T]{
		Kind:     kind,
		Expected: expected,
		Actual:   actual,
	}
}

// Error describes the failed numeric bound and the supplied value.
func (e *ErrNumConstraint[T]) Error() string {
	switch e.Kind {
	case NumConstraintMin:
		return fmt.Sprintf("number must be at least %v, got %v", e.Expected, e.Actual)
	case NumConstraintMax:
		return fmt.Sprintf("number must be at most %v, got %v", e.Expected, e.Actual)
	}

	return fmt.Sprintf("expected %v, got %v", e.Expected, e.Actual)
}
