package validators

import (
	"fmt"
	"regexp"
	"unicode/utf8"
)

type stringValidator struct {
	baseValidator[string]
}

func (s *stringValidator) addCheck(name string, arg any) error {
	switch name {
	case "min-length":
		return s.addMinLength(name, arg)
	case "max-length":
		return s.addMaxLength(name, arg)
	case "re":
		return s.addRegexp(name, arg)
	}

	return fmt.Errorf("unknown validator %s", name)
}

func (s *stringValidator) Validate(value string) error {
	return s.validate(value, func() (string, error) {
		return value, nil
	})
}

func (s *stringValidator) addMinLength(name string, arg any) error {
	length, ok := arg.(int64)
	if !ok || length < 0 {
		return fmt.Errorf("%s must be a non-negative integer", name)
	}

	s.checks = append(s.checks, func(val string) error {
		if utf8.RuneCountInString(val) < int(length) {
			return fmt.Errorf("%s must have at least %d characters", name, length)
		}
		return nil
	})

	return nil
}

func (s *stringValidator) addMaxLength(name string, arg any) error {
	length, ok := arg.(int64)
	if !ok || length < 0 {
		return fmt.Errorf("%s must be a non-negative integer", name)
	}

	s.checks = append(s.checks, func(val string) error {
		if utf8.RuneCountInString(val) > int(length) {
			return fmt.Errorf("%s must have at most %d characters", name, length)
		}
		return nil
	})

	return nil
}

func (s *stringValidator) addRegexp(name string, arg any) error {
	exprStr, ok := arg.(string)
	if !ok {
		return fmt.Errorf("%s must be a string", name)
	}

	expr, err := regexp.Compile(exprStr)
	if err != nil {
		return fmt.Errorf("%s is invalid regexp: %w", name, err)
	}

	s.checks = append(s.checks, func(val string) error {
		if !expr.MatchString(val) {
			return fmt.Errorf("%s does not match %s", name, exprStr)
		}
		return nil
	})

	return nil
}

func newStringValidator(properties map[string]any) (Validator, error) {
	rv := &stringValidator{
		validatorType: "str",
	}

	for k, v := range properties {
		if err := rv.addCheck(k, v); err != nil {
			return nil, err
		}
	}

	return rv, nil
}
