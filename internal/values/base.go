package values

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

type baseValue[T any] struct {
	prepare       func(string) (T, error)
	complete      func(string) ([]string, cobra.ShellCompDirective)
	checks        map[string]func(T) error
	validatorType string
}

func (b *baseValue[T]) Validate(value string) error {
	prepared, err := b.prepare(value)
	if err != nil {
		return err
	}

	for _, check := range b.checks {
		if err := check(prepared); err != nil {
			return err
		}
	}

	return nil
}

func (b *baseValue[T]) String() string {
	return fmt.Sprintf(
		"%s(checks=%s)",
		b.validatorType,
		strings.Join(slices.Sorted(maps.Keys(b.checks)), ", "),
	)
}

func (b *baseValue[T]) Complete(value string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if err := b.Validate(value); err == nil {
		return []cobra.Completion{value}, cobra.ShellCompDirectiveNoFileComp
	}

	values, directive := b.complete(value)
	values = slices.Clone(values)

	if directive&cobra.ShellCompDirectiveKeepOrder != 0 {
		seen := make(map[string]bool, len(values))
		values = slices.DeleteFunc(values, func(value string) bool {
			duplicate := seen[value]
			seen[value] = true

			return duplicate
		})
	} else {
		slices.Sort(values)
		values = slices.Compact(values)
	}

	return values, directive
}

func noopComplete(_ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
