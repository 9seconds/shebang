package validators

import (
	"errors"
	"fmt"
)

type Validator interface {
	AddCheck(string, any) error
	Validate(string) error
}

type baseValidator[T any] struct {
	checks []func(item T) error
}

func (b baseValidator[T]) validate(value string, prepare func() (T, error)) error {
	converted, err := prepare()
	if err != nil {
		return fmt.Errorf("cannot convert %s to %T", value, *new(T))
	}

	errs := []error{}
	for _, check := range b.checks {
		if err := check(converted); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}
