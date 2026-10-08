package validators

import (
	"fmt"
)

type Validator interface {
	Validate(string) error
	String() string
}

type baseValidator[T any] struct {
	validatorType string
	checks []func(item T) error
}

func (b baseValidator[T]) validate(value string, prepare func() (T, error)) error {
	converted, err := prepare()
	if err != nil {
		return fmt.Errorf("cannot convert %s to %T", value, *new(T))
	}

	for _, check := range b.checks {
		if err := check(converted); err != nil {
			return err
		}
	}

	return nil
}

func (b *baseValidator[T]) String() string {
	return fmt.Sprintf("%s(checks=%d)", b.validatorType, len(b.checks))
}

func New(valueType string, properties map[string]any) (Validator, error) {
	switch valueType {
	case "str":
		return newStringValidator(properties)
	}

	return nil, fmt.Errorf("unknown validator type %s", valueType)
}
