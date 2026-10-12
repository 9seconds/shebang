package values

import (
	"errors"
	"fmt"
	"slices"
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
func NewEnum(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueEnum{
		baseValue: newBase[string]("enum", len(subvalidators)),
	}
	val.toValue = func(input string) (string, error) {
		return input, nil
	}
	val.toComplete = val.genCompletions

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", k, err)
		}
	}

	val.setSubvalidator("choices", "", func(input string) error {
		if !val.choices[input] {
			return ErrUnknownChoice
		}

		return nil
	})

	return val, nil
}

func (v *valueEnum) addSubvalidator(name string, properties []any) error {
	if name == "choices" {
		return v.addSubvalidatorChoices(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueEnum) addSubvalidatorChoices(properties []any) error {
	choices, err := utils.All[string](properties)
	if err != nil {
		return err
	}

	slices.Sort(choices)
	v.subvalidatorDetails["choices"] = strings.Join(choices, ",")
	v.choices = make(map[string]bool, len(choices))

	for _, choice := range choices {
		v.choices[choice] = true
	}

	return nil
}

func (v *valueEnum) genCompletions(input string) ([]string, cobra.ShellCompDirective) {
	choices := make([]string, 0, len(v.choices))

	for choice := range v.choices {
		if strings.HasPrefix(choice, input) {
			choices = append(choices, choice)
		}
	}

	return choices, cobra.ShellCompDirectiveNoFileComp
}
