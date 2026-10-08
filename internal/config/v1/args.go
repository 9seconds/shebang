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
	if err := a.validateCount(len(args)); err != nil {
		return err
	}

	// Skip the leading group and reserve len(a.last) values at the end. The
	// middle slice contains only additional vararg items; the final slice
	// contains the trailing arguments. For first = [SOURCE, ITEMS1],
	// last = [DEST], and args = [source, item1, item2, dest], the slices are
	// rest = [item2] and last = [dest]. Empty groups produce empty slices.
	rest := args[len(a.first) : len(args)-len(a.last)]
	last := args[len(args)-len(a.last):]

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

func (a *argValidators) validateCount(count int) error {
	// Leading and trailing arguments are mandatory. The leading group also
	// includes required vararg items. Check their combined size before slicing
	// so the leading and trailing groups cannot overlap.
	//
	// Example:
	// first = [SOURCE, ITEMS1], last = [DEST]
	// At least three arguments are required.
	required := len(a.first) + len(a.last)
	if count < required {
		return NewArgumentCountError(ArgumentCountMinimum, required, count)
	}

	extra := count - required

	if a.varArg == nil {
		if extra > 0 {
			return NewArgumentCountError(ArgumentCountUnexpectedVarArgs, 0, extra)
		}

		return nil
	}

	// Required vararg items are already included in required. Add them back
	// to extra to obtain the total variadic count.
	//
	// Example:
	// first = [SOURCE, ITEMS1, ITEMS2], last = [DEST], count = 6
	// extra = 2; with minCount = 2, the variadic count is 4.
	variable := a.varArg.minCount + extra
	if a.varArg.maxCount >= 0 && variable > a.varArg.maxCount {
		return NewArgumentCountError(
			ArgumentCountMaximumVarArgs,
			a.varArg.maxCount,
			variable,
		)
	}

	return nil
}

func newArgValidators(conf *Config) (*argValidators, error) {
	validators := &argValidators{}

	first, err := newFixedArgValidators(conf.FirstArgs)
	if err != nil {
		return nil, err
	}

	validators.first = first

	if conf.VarArgs != nil {
		required, variable, err := newVarArgValidators(conf.VarArgs)
		if err != nil {
			return nil, err
		}

		validators.first = append(validators.first, required...)
		validators.varArg = variable
	}

	last, err := newFixedArgValidators(conf.LastArgs)
	if err != nil {
		return nil, err
	}

	validators.last = last

	return validators, nil
}

func newFixedArgValidators(args []Arg) ([]argValidator, error) {
	var validators []argValidator

	for _, arg := range args {
		name := strings.ToUpper(arg.Name)

		value, err := values.New(arg.Type, arg.Properties)
		if err != nil {
			return nil, fmt.Errorf("invalid argument %s: %w", name, err)
		}

		validators = append(validators, argValidator{
			name:      name,
			validator: value,
		})
	}

	return validators, nil
}

func newVarArgValidators(arg *VarArg) ([]argValidator, *varArgValidator, error) {
	name := strings.ToUpper(arg.Name)

	var required []argValidator

	if arg.MinCount != nil {
		value, err := values.New(arg.Type, arg.Properties)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid vararg type: %w", err)
		}

		for i := 1; i <= int(*arg.MinCount); i++ {
			required = append(required, argValidator{
				name:      fmt.Sprintf("%s%d", name, i),
				validator: value,
			})
		}
	}

	value, err := values.New(arg.Type, arg.Properties)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid argument %s: %w", name, err)
	}

	minCount := 0
	if arg.MinCount != nil {
		minCount = int(*arg.MinCount)
	}

	maxCount := -1
	if arg.MaxCount != nil {
		maxCount = int(*arg.MaxCount)
	}

	variable := &varArgValidator{
		name:      name,
		validator: value,
		minCount:  minCount,
		maxCount:  maxCount,
	}

	return required, variable, nil
}
