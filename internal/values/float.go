package values

import (
	"fmt"
	"math"
	"strconv"

	"github.com/9seconds/shebang/internal/utils"
)

type valueFloat struct {
	baseValue[float64]
}

// NewFloat constructs a finite float64 validator with inclusive min and max bounds.
//
//nolint:ireturn // Floating-point validators expose the shared Value interface.
func NewFloat(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueFloat{
		baseValue: newBase[float64]("float", len(subvalidators)),
	}
	val.toValue = func(input string) (float64, error) {
		value, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return 0, err
		}

		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, ErrFloatNonFinite
		}

		return value, nil
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

func (v *valueFloat) addSubvalidator(name string, properties []any) error {
	switch name {
	case "min":
		return v.addSubvalidatorMin(properties)
	case "max":
		return v.addSubvalidatorMax(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueFloat) addSubvalidatorMin(properties []any) error {
	limit, err := utils.One[float64](properties)
	if err != nil {
		return err
	}

	if math.IsNaN(limit) || math.IsInf(limit, 0) {
		return ErrFloatNonFinite
	}

	v.setSubvalidator("min", fmt.Sprintf("%g", limit), func(input float64) error {
		if input < limit {
			return NewErrNumConstraint(NumConstraintMin, limit, input)
		}

		return nil
	})

	return nil
}

func (v *valueFloat) addSubvalidatorMax(properties []any) error {
	limit, err := utils.One[float64](properties)
	if err != nil {
		return err
	}

	if math.IsNaN(limit) || math.IsInf(limit, 0) {
		return ErrFloatNonFinite
	}

	v.setSubvalidator("max", fmt.Sprintf("%g", limit), func(input float64) error {
		if input > limit {
			return NewErrNumConstraint(NumConstraintMax, limit, input)
		}

		return nil
	})

	return nil
}
