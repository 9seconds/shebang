package env

import (
	"os"
	"strings"

	"github.com/9seconds/shebang/internal/log"
)

const PREFIX = "SHEBANG_"

var (
	debugModeValue string = func() string {
		value := os.Getenv(PREFIX + "DEBUG")
		_ = os.Unsetenv(PREFIX + "DEBUG")
		return value
	}()
)

func Var(name string) string {
	if !strings.HasPrefix(name, PREFIX) {
		name = PREFIX + name
	}

	return strings.ToUpper(name)
}

func IsDebug() bool {
	return debugModeValue != ""
}

func Set(name string, value string) {
	name = Var(name)

	if err := os.Setenv(name, value); err != nil {
		log.Die("Cannot set environment variable %s to %s: %s", name, value, err)
	}

	log.Print("Set %s to %s", name, value)
}
