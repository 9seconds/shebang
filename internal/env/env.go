// Package env manages shebang environment variables and startup debug state.
package env

import (
	"os"
	"strings"

	"github.com/9seconds/shebang/internal/log"
)

// Prefix identifies environment variables owned by shebang.
const Prefix = "SHEBANG_"

var debugModeValue = func() string {
	value := os.Getenv(Prefix + "DEBUG")
	_ = os.Unsetenv(Prefix + "DEBUG")

	return value
}()

// Var adds Prefix when absent, replaces hyphens with underscores, and converts
// the resulting name to uppercase.
func Var(name string) string {
	if !strings.HasPrefix(name, Prefix) {
		name = Prefix + name
	}

	return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

// IsDebug reports whether SHEBANG_DEBUG was nonempty at process startup.
// The startup variable is removed from the environment during initialization.
func IsDebug() bool {
	return debugModeValue != ""
}

// Set exports value under the normalized name, exiting if the variable is invalid.
func Set(name string, value string) {
	name = Var(name)

	if err := os.Setenv(name, value); err != nil {
		log.Dief("Cannot set environment variable %s to %s: %s", name, value, err)
	}

	log.Printf("Set %s to %s", name, value)
}
