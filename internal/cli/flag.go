package cli

import "github.com/9seconds/shebang/internal/env"

const (
	PREFIX_FLAG_LONG = env.PREFIX + "FL_"
	PREFIX_FLAG_SHORT = env.PREFIX + "FS_"
)

type Flag struct {
	Value bool
	Long  string
	Short string
}

func (f *Flag) SetEnv() {
	if !f.Value {
		return
	}

	env.Set(PREFIX_FLAG_LONG + f.Long, "true")

	if f.Short != "" {
		env.Set(PREFIX_FLAG_SHORT + f.Short, "true")
	}
}
