package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/9seconds/shebang/internal/validators"
)

type Option struct {
	values []string
	name string
	optionType string
	minCount int
	maxCount int
	validator validators.Validator
}

func (f *Option) String() string {
	elements := []string{
		fmt.Sprintf("name=%s", f.name),
		fmt.Sprintf("optionType=%s", f.optionType),
		fmt.Sprintf("minCount=%d", f.minCount),
		fmt.Sprintf("maxCount=%d", f.maxCount),
		fmt.Sprintf("validator=%s", f.validator.String()),
	}
	return strings.Join(elements, "\n")
}

func (f *Option) Type() string {
	return f.optionType
}

func (f *Option) Set(value string) error {
	if err := f.validator.Validate(value); err != nil {
		return err
	}

	f.values = append(f.values, value)

	return nil
}

func (f *Option) Validate() error {
	if lv := len(f.values); lv < f.minCount {
		return fmt.Errorf("there must be at least %d elements, have only %d", f.minCount, lv)
	}

	if lv := len(f.values); lv > f.maxCount {
		return fmt.Errorf("there must be at most %d elements, have only %d", f.maxCount, lv)
	}

	return nil
}

func (f *Option) SetEnv() error {
	return os.Setenv(Env(f.name), strings.Join(f.values, "\x00"))
}

func NewOption(name, optionType string, minCount, maxCount int, validator validators.Validator) *Option {
	return &Option{
		name: name,
		optionType: optionType,
		minCount: minCount,
		maxCount: maxCount,
		validator: validator,
	}
}
