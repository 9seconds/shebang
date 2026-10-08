package cli

import (
	"fmt"
	"strings"

	"github.com/9seconds/shebang/internal/env"
	"github.com/9seconds/shebang/internal/validators"
)

type Option struct {
	values []string

	OptionType string
	Name       string
	MinCount   int
	MaxCount   int
	Validator  validators.Validator
}

func (f *Option) String() string {
	return strings.Join(f.values, "\x00")
}

func (f *Option) Repr() string {
	elements := []string{
		fmt.Sprintf("name=%s", f.Name),
		fmt.Sprintf("values=%v", f.values),
		fmt.Sprintf("optionType=%s", f.OptionType),
		fmt.Sprintf("minCount=%d", f.MinCount),
		fmt.Sprintf("maxCount=%d", f.MaxCount),
		fmt.Sprintf("validator=%s", f.Validator.String()),
	}
	return strings.Join(elements, "\n")
}

func (f *Option) SetEnv() {
	if len(f.values) > 0 {
		env.Set(f.Name, f.String())
	}
}

func (f *Option) Type() string {
	return f.OptionType
}

func (f *Option) Set(value string) error {
	if err := f.Validator.Validate(value); err != nil {
		return err
	}

	f.values = append(f.values, value)

	return nil
}

func (f *Option) Validate() error {
	if lv := len(f.values); lv < f.MinCount {
		return fmt.Errorf("there must be at least %d elements, have only %d", f.MaxCount, lv)
	}

	if lv := len(f.values); lv > f.MaxCount {
		return fmt.Errorf("there must be at most %d elements, have only %d", f.MaxCount, lv)
	}

	return nil
}
