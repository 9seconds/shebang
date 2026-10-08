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
	complete      func(T) ([]string, cobra.ShellCompDirective)
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
	prepared, err := b.prepare(value)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return b.complete(prepared)
}

func noopComplete(_ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
