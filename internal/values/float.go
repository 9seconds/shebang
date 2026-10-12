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
func NewFloat(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueFloat{
		baseValue: newBase[float64]("float", len(subvalidators)),
	}
	val.toValue = func(input string) (float64, error){
			value, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return 0, err
			}

			if math.IsNaN(value) || math.IsInf(value, 0) {
				return 0, ErrNonFiniteFloat
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
		return v.addMin(properties)
	case "max":
		return v.addMax(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueFloat) addMin(properties []any) error {
	limit, err := utils.One[float64](properties)
	if err != nil {
		return err
	}

	if math.IsNaN(limit) || math.IsInf(limit, 0) {
		return ErrNonFiniteFloat
	}

	v.setSubvalidator("min", fmt.Sprintf("%g", limit), func(input float64) error {
		if input < limit {
			return NewNumConstraintError(NumConstraintMin, limit, input)
		}

		return nil
	})

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

	v.setSubvalidator("max", fmt.Sprintf("%g", limit), func(input float64) error {
		if input > limit {
			return NewNumConstraintError(NumConstraintMax, limit, input)
		}

		return nil
	})

	return nil
}
