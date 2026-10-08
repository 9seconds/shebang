// Package utils provides typed extraction of parsed configuration arguments.
package utils

import (
	"errors"
	"fmt"
)

// ErrEmpty indicates that a required argument is missing.
var ErrEmpty = errors.New("no elements were defined")

// ErrSingleArgumentExpected indicates that more than one argument was supplied.
var ErrSingleArgumentExpected = errors.New("expected 1 element")

// ArgumentTypeError describes an argument whose type differs from the expected type.
type ArgumentTypeError struct {
	Expected string
	Actual   string
}

// NewArgumentTypeError describes the types of expected and actual values.
func NewArgumentTypeError(expected, actual any) *ArgumentTypeError {
	return &ArgumentTypeError{
		Expected: fmt.Sprintf("%T", expected),
		Actual:   fmt.Sprintf("%T", actual),
	}
}

// Error describes the expected and actual argument types.
func (e *ArgumentTypeError) Error() string {
	return fmt.Sprintf("expected %s parameter, got %s", e.Expected, e.Actual)
}

// All extracts values of type T in order, returning an error on a type mismatch.
func All[T any](values []any) ([]T, error) {
	elements := make([]T, len(values))

	for idx, value := range values {
		val, ok := value.(T)
		if !ok {
			return nil, NewArgumentTypeError(*new(T), value)
		}

		elements[idx] = val
	}

	return elements, nil
}

// One requires exactly one value of type T. Empty input returns ErrEmpty.
//
//nolint:ireturn // Return the caller-selected generic type T.
func One[T any](values []any) (T, error) {
	var empty T

	converted, err := All[T](values)
	if err != nil {
		return empty, err
	}

	switch len(converted) {
	case 0:
		return empty, ErrEmpty
	case 1:
		return converted[0], nil
	}

	return empty, fmt.Errorf("%w, got %d", ErrSingleArgumentExpected, len(values))
}
