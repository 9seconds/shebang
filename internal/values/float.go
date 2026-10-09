package values

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/9seconds/shebang/internal/utils"
)

// ErrNonFiniteFloat indicates NaN or an infinite float value or bound.
var ErrNonFiniteFloat = errors.New("float must be finite")

type valueFloat struct {
	baseValue[float64]
}

// NewFloat constructs a finite float64 validator with inclusive min and max bounds.
//
//nolint:ireturn // Floating-point validators expose the shared Value interface.
func NewFloat(properties map[string][]any) (Value, error) {
	val := &valueFloat{
		validatorType: "float",
		complete:      noopComplete,
		checks:        make(map[string]func(float64) error),
		prepare: func(v string) (float64, error) {
			value, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return 0, err
			}

			if math.IsNaN(value) || math.IsInf(value, 0) {
				return 0, ErrNonFiniteFloat
			}

			return value, nil
		},
	}

	for k, v := range properties {
		if err := val.addCheck(k, v); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueFloat) addCheck(name string, value []any) error {
	switch name {
	case "min":
		return v.addMin(value)
	case "max":
		return v.addMax(value)
	}

	return ErrUnknownProperty
}

func (v *valueFloat) addMin(properties []any) error {
	limit, err := utils.One[float64](properties)
	if err != nil {
		return err
	}

	if math.IsNaN(limit) || math.IsInf(limit, 0) {
		return ErrNonFiniteFloat
	}

	name := fmt.Sprintf("min:%g", limit)

	v.checks[name] = func(value float64) error {
		if value < limit {
			return NewNumConstraintError(NumConstraintMin, limit, value)
		}

		return nil
	}

	return nil
}

func (v *valueFloat) addMax(properties []any) error {
	limit, err := utils.One[float64](properties)
	if err != nil {
		return err
	}

	if math.IsNaN(limit) || math.IsInf(limit, 0) {
		return ErrNonFiniteFloat
	}

	name := fmt.Sprintf("max:%g", limit)

	v.checks[name] = func(value float64) error {
		if value > limit {
			return NewNumConstraintError(NumConstraintMax, limit, value)
		}

		return nil
	}

	return nil
}
