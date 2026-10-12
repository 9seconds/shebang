package values

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

type baseValue[T any] struct {
	valueType string
	subvalidators map[string]func(T) error
	subvalidatorDetails map[string]string
	toValue func(string) (T, error)
	toComplete func(string) ([]string, cobra.ShellCompDirective)
}

func (b *baseValue[T]) Validate(value string) error {
	prepared, err := b.toValue(value)
	if err != nil {
		return err
	}

	for _, subvalidator := range b.subvalidators {
		if err := subvalidator(prepared); err != nil {
			return err
		}
	}

	return nil
}

func (b *baseValue[T]) String() string {
	details := make([]string, 0, len(b.subvalidatorDetails))

	for name, detail := range b.subvalidatorDetails {
		if detail == "" {
			details = append(details, name)
		} else {
			details = append(details, name + ":" + detail)
		}
	}

	slices.Sort(details)

	return fmt.Sprintf("%s(%s)", b.valueType, strings.Join(details, ", "))
}

func (b *baseValue[T]) Complete(value string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if err := b.Validate(value); err == nil {
		return []cobra.Completion{value}, cobra.ShellCompDirectiveNoFileComp
	}

	values, directive := b.toComplete(value)

	// shell may change the order though, this is fine
	slices.Sort(values)
	values = slices.Compact(values)

	return values, directive
}

func (b *baseValue[T]) setSubvalidator(name, detail string, validator func(T) error) {
	b.subvalidatorDetails[name] =  detail
	b.subvalidators[name] = validator
}

func newBase[T any](valueType string, subvalidatorsLen int) baseValue[T] {
	return baseValue[T]{
		valueType: valueType,
		subvalidators: make(map[string]func(T) error, subvalidatorsLen),
		subvalidatorDetails: make(map[string]string, subvalidatorsLen),
		toComplete: func(_ string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
}
