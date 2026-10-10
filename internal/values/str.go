package values

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/9seconds/shebang/internal/utils"
)

type valueStr struct {
	baseValue[string]
}

// NewStr constructs a string validator with rune-length and regular-expression
// subvalidators described by properties.
//
//nolint:ireturn // String validators expose the shared Value interface.
func NewStr(properties map[string][]any) (Value, error) {
	val := &valueStr{
		validatorType: "str",
		complete:      noopComplete,
		subvalidators: make(map[string]func(string) error),
		prepare: func(v string) (string, error) {
			return v, nil
		},
	}

	for k, v := range properties {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueStr) addSubvalidator(name string, value []any) error {
	switch name {
	case "min-length":
		return v.addMinLength(value)
	case "max-length":
		return v.addMaxLength(value)
	case "re":
		return v.addRe(value)
	}

	return ErrUnknownProperty
}

func (v *valueStr) addMinLength(properties []any) error {
	length, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if length < 0 {
		return NewNegativeLengthError(length)
	}

	intLength := int(length)
	name := fmt.Sprintf("min-length:%d", intLength)

	v.subvalidators[name] = func(value string) error {
		if lv := utf8.RuneCountInString(value); lv < intLength {
			return NewLengthConstraintError(LengthConstraintMinimum, intLength, lv)
		}

		return nil
	}

	return nil
}

func (v *valueStr) addMaxLength(properties []any) error {
	length, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if length < 0 {
		return NewNegativeLengthError(length)
	}

	intLength := int(length)
	name := fmt.Sprintf("max-length:%d", intLength)

	v.subvalidators[name] = func(value string) error {
		if lv := utf8.RuneCountInString(value); lv > intLength {
			return NewLengthConstraintError(LengthConstraintMaximum, intLength, lv)
		}

		return nil
	}

	return nil
}

func (v *valueStr) addRe(properties []any) error {
	strExpr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	expression, err := regexp.Compile(strExpr)
	if err != nil {
		return fmt.Errorf("incorrect regular expression %s: %w", strExpr, err)
	}

	name := "re:" + strExpr

	v.subvalidators[name] = func(value string) error {
		if !expression.MatchString(value) {
			return NewRegexMismatchError(value, strExpr)
		}

		return nil
	}

	return nil
}
