package values

import (
	"fmt"
	"strconv"

	"github.com/9seconds/shebang/internal/utils"
)

type valueInt struct {
	baseValue[int64]
}

// NewInt constructs a signed decimal int64 validator with inclusive min and max bounds.
//
//nolint:ireturn // Integer validators expose the shared Value interface.
func NewInt(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueInt{
		baseValue: newBase[int64]("int", len(subvalidators)),
	}
	val.toValue = func(input string) (int64, error) {
		return strconv.ParseInt(input, 10, 64)
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

func (v *valueInt) addSubvalidator(name string, properties []any) error {
	switch name {
	case "min":
		return v.addMin(properties)
	case "max":
		return v.addMax(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueInt) addMin(properties []any) error {
	limit, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("min", strconv.FormatInt(limit, 10), func(input int64) error {
		if input < limit {
			return NewNumConstraintError(NumConstraintMin, limit, input)
		}

		return nil
	})

	return nil
}

func (v *valueInt) addMax(properties []any) error {
	limit, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("max", strconv.FormatInt(limit, 10), func(input int64) error {
		if input > limit {
			return NewNumConstraintError(NumConstraintMax, limit, input)
		}

		return nil
	})

	return nil
}
