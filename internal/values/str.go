package values

import (
	"fmt"
	"regexp"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/9seconds/shebang/internal/utils"
)

type valueStr struct {
	baseValue[string]
}

// NewStr constructs a string validator with rune-length and regular-expression
// subvalidators described by subvalidators.
//
//nolint:ireturn // String validators expose the shared Value interface.
func NewStr(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueStr{
		baseValue: newBase[string]("str", len(subvalidators)),
	}
	val.toValue = func(input string) (string, error) {
		for _, char := range input {
			if !unicode.IsPrint(char) {
				return "", ErrStrNotPrintable
			}
		}

		return input, nil
	}

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueStr) addSubvalidator(name string, properties []any) error {
	switch name {
	case "min-length":
		return v.addMinLength(properties)
	case "max-length":
		return v.addMaxLength(properties)
	case "re":
		return v.addRe(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueStr) addMinLength(properties []any) error {
	length, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if length < 0 {
		return NewErrStrNegativeLength(length)
	}

	intLength := int(length)

	v.setSubvalidator("min-length", strconv.Itoa(intLength), func(input string) error {
		if lv := utf8.RuneCountInString(input); lv < intLength {
			return NewErrStrLengthConstraint(StrLengthConstraintMinimum, intLength, lv)
		}

		return nil
	})

	return nil
}

func (v *valueStr) addMaxLength(properties []any) error {
	length, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if length < 0 {
		return NewErrStrNegativeLength(length)
	}

	intLength := int(length)

	v.setSubvalidator("max-length", strconv.Itoa(intLength), func(input string) error {
		if lv := utf8.RuneCountInString(input); lv > intLength {
			return NewErrStrLengthConstraint(StrLengthConstraintMaximum, intLength, lv)
		}

		return nil
	})

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

	v.setSubvalidator("re", strExpr, func(input string) error {
		if !expression.MatchString(input) {
			return NewErrStrRegexMismatch(input, strExpr)
		}

		return nil
	})

	return nil
}
