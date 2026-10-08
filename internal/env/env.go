package env

import (
	"os"
	"strings"

	"github.com/9seconds/shebang/internal/log"
)

const PREFIX = "SHEBANG_"

func Var(name string) string {
	if strings.HasPrefix(name, PREFIX) {
		return name
	}

	return PREFIX + strings.ToUpper(name)
}

func Set(name string, value string) {
	if err := os.Setenv(Var(name), value); err != nil {
		log.Die("Cannot set environment variable %s to %s: %s", name, value, err)
	}
}
