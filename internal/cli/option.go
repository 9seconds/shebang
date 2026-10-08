package cli

import (
	"fmt"

	"github.com/9seconds/shebang/internal/env"
	"github.com/9seconds/shebang/internal/values"
)

const (
	// PrefixOptLong prefixes environment variables for long option names.
	PrefixOptLong = env.Prefix + "OL_"
	// PrefixOptShort prefixes environment variables for short option names.
	PrefixOptShort = env.Prefix + "OS_"
)

// Option stores a validated string option and its command-line names.
type Option struct {
	Long      string
	Short     string
	ValueType string
	Value     *string
	Validator values.Value
}

// String returns an empty default value for command-line help.
func (o *Option) String() string {
	return ""
}

// AsString describes the option names and validator for debug logging.
func (o *Option) AsString() string {
	return fmt.Sprintf("%s %s (value=%s)", o.Long, o.Short, o.Validator)
}

// Type returns the value type displayed in command-line help.
func (o *Option) Type() string {
	return o.ValueType
}

// Set validates value before replacing the stored option value.
func (o *Option) Set(value string) error {
	if err := o.Validator.Validate(value); err != nil {
		return err
	}

	o.Value = &value

	return nil
}

// SetEnv exports a set option under its long and optional short names.
func (o *Option) SetEnv() {
	if o.Value == nil {
		return
	}

	env.Set(PrefixOptLong+o.Long, *o.Value)

	if o.Short != "" {
		env.Set(PrefixOptShort+o.Short, *o.Value)
	}
}
