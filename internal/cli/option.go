package cli

import (
	"fmt"

	"github.com/9seconds/shebang/internal/env"
	"github.com/9seconds/shebang/internal/values"
)

const (
	PREFIX_OPT_LONG = env.PREFIX + "OL_"
	PREFIX_OPT_SHORT = env.PREFIX + "OS_"
)

type Option struct {
	Long string
	Short string
	ValueType string
	Value *string
	Validator values.Value
}

func (o *Option) String() string {
	return ""
}

func (o *Option) AsString() string {
	return fmt.Sprintf("%s %s (value=%s)", o.Long, o.Short, o.Validator)
}

func (o *Option) Type() string {
	return o.ValueType
}

func (o *Option) Set(value string) error {
	if err := o.Validator.Validate(value); err != nil {
		return err
	}

	o.Value = &value

	return nil
}

func (o *Option) SetEnv() {
	if o.Value == nil {
		return
	}

	env.Set(PREFIX_OPT_LONG + o.Long, *o.Value)

	if o.Short != "" {
		env.Set(PREFIX_OPT_SHORT + o.Short, *o.Value)
	}
}
