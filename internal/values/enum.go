package values

import (
	"errors"
	"fmt"
	"strings"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/spf13/cobra"
)

// ErrUnknownChoice indicates a value outside the configured enum choices.
var ErrUnknownChoice = errors.New("unknown choice")

type valueEnum struct {
	baseValue[string]

	choices map[string]bool
}

// NewEnum constructs an exact, case-sensitive choice validator with prefix completion.
//
//nolint:ireturn // Enum validators expose the shared Value interface.
func NewEnum(properties map[string][]any) (Value, error) {
	val := &valueEnum{
		validatorType: "enum",
		subvalidators: make(map[string]func(string) error, len(properties)),
		prepare: func(v string) (string, error) {
			return v, nil
		},
	}
	val.complete = val.completeChoices

	for k, v := range properties {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", k, err)
		}
	}

	val.subvalidators["choices"] = func(v string) error {
		if !val.choices[v] {
			return ErrUnknownChoice
		}

		return nil
	}

	return val, nil
}

func (v *valueEnum) addSubvalidator(name string, value []any) error {
	if name == "choices" {
		return v.addSubvalidatorChoices(value)
	}

	return ErrUnknownProperty
}

func (v *valueEnum) addSubvalidatorChoices(value []any) error {
	choices, err := utils.All[string](value)
	if err != nil {
		return err
	}

	v.choices = make(map[string]bool, len(choices))

	for _, choice := range choices {
		v.choices[choice] = true
	}

	return nil
}

func (v *valueEnum) completeChoices(input string) ([]cobra.Completion, cobra.ShellCompDirective) {
	choices := make([]cobra.Completion, 0, len(v.choices))

	for choice := range v.choices {
		if strings.HasPrefix(choice, input) {
			choices = append(choices, choice)
		}
	}

	return choices, cobra.ShellCompDirectiveNoFileComp
}
