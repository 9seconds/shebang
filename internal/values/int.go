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
func NewInt(properties map[string][]any) (Value, error) {
	val := &valueInt{
		validatorType: "int",
		complete:      noopComplete,
		checks:        make(map[string]func(int64) error, len(properties)),
		prepare: func(v string) (int64, error) {
			return strconv.ParseInt(v, 10, 64)
		},
	}

	for k, v := range properties {
		if err := val.addCheck(k, v); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueInt) addCheck(name string, value []any) error {
	switch name {
	case "min":
		return v.addMin(value)
	case "max":
		return v.addMax(value)
	}

	return ErrUnknownProperty
}

func (v *valueInt) addMin(properties []any) error {
	limit, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("min:%d", limit)

	v.checks[name] = func(value int64) error {
		if value < limit {
			return NewNumConstraintError(NumConstraintMin, limit, value)
		}

		return nil
	}

	return nil
}

func (v *valueInt) addMax(properties []any) error {
	limit, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("max:%d", limit)

	v.checks[name] = func(value int64) error {
		if value > limit {
			return NewNumConstraintError(NumConstraintMax, limit, value)
		}

		return nil
	}

	return nil
}
