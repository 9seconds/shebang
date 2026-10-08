package v1

import (
	"fmt"
	"strings"

	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
)

type argValidator struct {
	name      string
	validator values.Value
}

type varArgValidator struct {
	argValidator
	minCount int
	maxCount int
}

type argValidators struct {
	first  []argValidator
	last   []argValidator
	varArg *varArgValidator
}

func (a *argValidators) Validate(args []string) error {
	// Explanation:
	// Leading and trailing arguments are mandatory. The leading group also
	// includes the minimum required vararg items added by newArgValidators.
	// Check their combined size before slicing args so neither group is missing
	// and the leading and trailing slices cannot overlap.
	//
	// Example:
	// a.first = [SOURCE, ITEMS1]
	// a.last = [DEST]
	// args = [source, item1, item2, dest]
	// At least three arguments are required. SOURCE and ITEMS1 consume the first
	// two, DEST consumes the last, and item2 is an optional vararg item.
	if sz := len(a.first) + len(a.last); len(args) < sz {
		return fmt.Errorf("there must be at least %d arguments, got %d", sz, len(args))
	}

	// Skip the leading group and reserve len(a.last) values at the end. The
	// middle slice contains only additional vararg items; the final slice
	// contains the trailing arguments. In the example above, rest = [item2]
	// and last = [dest]. Empty groups produce valid empty slices.
	rest := args[len(a.first) : len(args)-len(a.last)]
	last := args[len(args)-len(a.last):]

	if a.varArg == nil && len(rest) > 0 {
		return fmt.Errorf("expected 0 variadic arguments, got %d", len(rest))
	}

	if a.varArg != nil {
		count := a.varArg.minCount + len(rest)

		// Explanation:
		// Required vararg items are already in a.first, so add minCount to the
		// remaining items to recover the total vararg count. A negative maximum
		// means unlimited; a nonnegative maximum includes the required items.
		//
		// Example:
		// a.first = [SOURCE, ITEMS1, ITEMS2]
		// a.last = [DEST]
		// args = [source, item1, item2, item3, item4, dest]
		// With minCount = 2, rest has two items and count = 4. A maxCount of 3
		// rejects this input even though only two items are in rest.
		if a.varArg.maxCount >= 0 && count > a.varArg.maxCount {
			return fmt.Errorf(
				"there must be at most %d variadic arguments, got %d",
				a.varArg.maxCount,
				count,
			)
		}
	}

	for idx, arg := range a.first {
		if err := arg.validator.Validate(args[idx]); err != nil {
			return fmt.Errorf("invalid argument %s: %w", arg.name, err)
		}
	}

	for idx, arg := range a.last {
		if err := arg.validator.Validate(last[idx]); err != nil {
			return fmt.Errorf("invalid argument %s: %w", arg.name, err)
		}
	}

	for idx, argValue := range rest {
		if err := a.varArg.validator.Validate(argValue); err != nil {
			return fmt.Errorf(
				"invalid argument %s%d: %w",
				a.varArg.name,
				a.varArg.minCount+idx+1,
				err,
			)
		}
	}

	return nil
}

func (a *argValidators) Complete(
	args []string,
	toComplete string,
) ([]cobra.Completion, cobra.ShellCompDirective) {
	// Compute the total positional capacity before choosing a validator.
	// a.first already contains the minimum required vararg items, so a bounded
	// group contributes only maxCount - minCount additional slots. A negative
	// vararg maximum makes the total capacity unlimited.
	maxArgs := len(a.first) + len(a.last)
	if a.varArg != nil {
		if a.varArg.maxCount < 0 {
			maxArgs = -1
		} else {
			maxArgs += a.varArg.maxCount - a.varArg.minCount
		}
	}

	switch {
	// args contains completed words; toComplete is the next word. Once all
	// available slots are filled, there is no positional value to complete.
	case maxArgs >= 0 && len(args) >= maxArgs:
		return nil, cobra.ShellCompDirectiveNoFileComp

	// Fill leading positions, including required vararg items, in order.
	case len(args) < len(a.first):
		return a.first[len(args)].validator.Complete(toComplete)

	// Once leading positions are filled, prefer trailing validators because
	// the split between optional varargs and trailing values is ambiguous
	// while typing.
	case len(a.last) > 0:
		// Advance through trailing validators, then reuse the final one as
		// additional words shift earlier values into the vararg group.
		idx := min(len(a.last)-1, len(args)-len(a.first))
		return a.last[idx].validator.Complete(toComplete)

	// Without trailing positions, all additional values belong to the vararg.
	case a.varArg != nil:
		return a.varArg.validator.Complete(toComplete)
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func newArgValidators(c *Config) (*argValidators, error) {
	rv := &argValidators{}

	for _, arg := range c.FirstArgs {
		name := strings.ToUpper(arg.Name)

		value, err := values.New(arg.Type, arg.Properties)
		if err != nil {
			return nil, fmt.Errorf("invalid argument %s: %w", name, err)
		}

		rv.first = append(rv.first, argValidator{
			name:      name,
			validator: value,
		})
	}

	if c.VarArgs != nil && c.VarArgs.MinCount != nil {
		value, err := values.New(c.VarArgs.Type, c.VarArgs.Properties)
		if err != nil {
			return nil, fmt.Errorf("invalid vararg type: %w", err)
		}

		for i := 1; i <= int(*c.VarArgs.MinCount); i++ {
			rv.first = append(rv.first, argValidator{
				name:      fmt.Sprintf("%s%d", strings.ToUpper(c.VarArgs.Name), i),
				validator: value,
			})
		}
	}

	if c.VarArgs != nil {
		name := strings.ToUpper(c.VarArgs.Name)

		value, err := values.New(c.VarArgs.Type, c.VarArgs.Properties)
		if err != nil {
			return nil, fmt.Errorf("invalid argument %s: %w", name, err)
		}

		minCount := 0
		if c.VarArgs.MinCount != nil {
			minCount = int(*c.VarArgs.MinCount)
		}

		maxCount := -1
		if c.VarArgs.MaxCount != nil {
			maxCount = int(*c.VarArgs.MaxCount)
		}

		rv.varArg = &varArgValidator{
			name:      name,
			validator: value,
			minCount:  minCount,
			maxCount:  maxCount,
		}
	}

	for _, arg := range c.LastArgs {
		name := strings.ToUpper(arg.Name)

		value, err := values.New(arg.Type, arg.Properties)
		if err != nil {
			return nil, fmt.Errorf("invalid argument %s: %w", name, err)
		}

		rv.last = append(rv.last, argValidator{
			name:      name,
			validator: value,
		})
	}

	return rv, nil
}
