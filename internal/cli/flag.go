package cli

import "strconv"

type Flag struct {
	Value bool
	Name  string
}

func (f *Flag) String() string {
	return strconv.FormatBool(f.Value)
}
