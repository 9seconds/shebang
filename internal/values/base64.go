package values

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/9seconds/shebang/internal/utils"
)

var base64Encodings = map[string]*base64.Encoding{
	"raw":     base64.RawStdEncoding,
	"raw-url": base64.RawURLEncoding,
	"std":     base64.StdEncoding,
	"url":     base64.URLEncoding,
}

type valueBase64 struct {
	baseValue[[]byte]

	encoding *base64.Encoding
}

// NewBase64 constructs a Base64 validator with decoded-byte length constraints.
//
//nolint:ireturn // Base64 validators expose the shared Value interface.
func NewBase64(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueBase64{
		baseValue: newBase[[]byte]("base64", len(subvalidators)),
		encoding:  base64.StdEncoding,
	}
	val.toValue = func(input string) ([]byte, error) {
		return val.encoding.DecodeString(input)
	}

	for k, v := range properties {
		if err := val.addProperty(k, v); err != nil {
			return nil, fmt.Errorf("incorrect property %s: %w", k, err)
		}
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("incorrect subvalidator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueBase64) addProperty(name string, property any) error {
	if name == "format" {
		return v.addPropertyFormat(property)
	}

	return ErrUnknownProperty
}

func (v *valueBase64) addPropertyFormat(property any) error {
	strValue, isString := property.(string)
	if !isString {
		return utils.NewArgumentTypeError("", property)
	}

	encoding, exists := base64Encodings[strValue]
	if !exists {
		return NewErrBase64Format(strValue)
	}

	v.encoding = encoding

	return nil
}

func (v *valueBase64) addSubvalidator(name string, properties []any) error {
	switch name {
	case "min-bytes":
		return v.addSubvalidatorMinBytes(properties)
	case "max-bytes":
		return v.addSubvalidatorMaxBytes(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueBase64) addSubvalidatorMinBytes(properties []any) error {
	intVal, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if intVal < 0 {
		return ErrBase64NegativeLength
	}

	v.setSubvalidator("min-bytes", strconv.FormatInt(intVal, 10), func(input []byte) error {
		if int64(len(input)) < intVal {
			return NewErrBase64LengthConstraint("at least", intVal, int64(len(input)))
		}

		return nil
	})

	return nil
}

func (v *valueBase64) addSubvalidatorMaxBytes(properties []any) error {
	intVal, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if intVal < 0 {
		return ErrBase64NegativeLength
	}

	v.setSubvalidator("max-bytes", strconv.FormatInt(intVal, 10), func(input []byte) error {
		if int64(len(input)) > intVal {
			return NewErrBase64LengthConstraint("at most", intVal, int64(len(input)))
		}

		return nil
	})

	return nil
}
