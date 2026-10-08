package cli

import "github.com/9seconds/shebang/internal/env"

const (
	// PrefixFlagLong prefixes environment variables for long flag names.
	PrefixFlagLong = env.Prefix + "FL_"
	// PrefixFlagShort prefixes environment variables for short flag names.
	PrefixFlagShort = env.Prefix + "FS_"
)

// Flag stores a boolean flag and its long and short names.
type Flag struct {
	Value bool
	Long  string
	Short string
}

// SetEnv exports enabled flags under their long and optional short names.
func (f *Flag) SetEnv() {
	if !f.Value {
		return
	}

	env.Set(PrefixFlagLong+f.Long, "true")

	if f.Short != "" {
		env.Set(PrefixFlagShort+f.Short, "true")
	}
}
