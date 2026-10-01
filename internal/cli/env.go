package cli

import "strings"

func Env(name string) string {
	return "SHEBANG_" + strings.ToUpper(name)
}
