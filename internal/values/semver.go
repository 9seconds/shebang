package values

import (
	"errors"
	"fmt"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/Masterminds/semver/v3"
)

type valueSemver struct {
	baseValue[*semver.Version]
}

// NewSemver constructs a semantic-version validator with optional version constraints.
//
//nolint:ireturn // Semantic-version validators expose the shared Value interface.
func NewSemver(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueSemver{
		baseValue: newBase[*semver.Version]("semver", len(subvalidators)),
	}
	val.toValue = semver.NewVersion

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("incorrect subvalidator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueSemver) addSubvalidator(name string, properties []any) error {
	if name == "constraint" {
		return v.addSubvalidatorConstraint(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueSemver) addSubvalidatorConstraint(properties []any) error {
	consStr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	cons, err := semver.NewConstraint(consStr)
	if err != nil {
		return fmt.Errorf("incorrect constraint: %w", err)
	}

	v.setSubvalidator("constraint", consStr, func(input *semver.Version) error {
		if ok, errs := cons.Validate(input); !ok {
			return errors.Join(errs...)
		}

		return nil
	})

	return nil
}
