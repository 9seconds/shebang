package utils

import (
	"errors"
	"fmt"
)

var (
	ErrEmpty = errors.New("no elements were defined")
)

func All[T any](values []any) ([]T, error) {
	elements := make([]T, len(values))

	for idx, value := range values {
		val, ok := value.(T)
		if !ok {
			return nil, fmt.Errorf("expected %T parameter, got %T", *new(T), value)
		}
		elements[idx] = val
	}

	return elements, nil
}

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

	return empty, fmt.Errorf("expected 1 element, got %d", len(values))
}
