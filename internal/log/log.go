// Package log provides optional debug output and fatal error reporting.
package log

import (
	"fmt"
	"io"
	"iter"
	"log"
	"os"
	"strings"
	"unicode"
)

var (
	main = log.New(io.Discard, ">>> ", 0)
	err  = log.New(os.Stderr, "", 0)
)

// Configure enables debug output on stderr when isDebug is true.
func Configure(isDebug bool) {
	if isDebug {
		main.SetOutput(os.Stderr)
	}
}

// Dief prints a formatted message to stderr and exits with status one.
func Dief(format string, arg ...any) {
	format = strings.TrimRightFunc(format, unicode.IsSpace) + "\n"
	err.Fatalf(format, arg...)
}

// Printf writes a formatted debug message.
func Printf(format string, arg ...any) {
	PrintVal("", fmt.Sprintf(format, arg...))
}

// PrintVal writes a labeled debug value, indenting continuation lines.
func PrintVal(reason string, value string) {
	value = strings.TrimSpace(value)

	reason = strings.TrimSpace(reason)
	if reason != "" && !strings.HasSuffix(reason, ":") {
		reason += ": "
	}

	emptyPrefix := strings.Repeat(" ", len(reason))
	currentPrefix := reason

	for line := range iterLines(value) {
		main.Println(currentPrefix, line)
		currentPrefix = emptyPrefix
	}
}

func iterLines(value string) iter.Seq[string] {
	value = strings.TrimRightFunc(value, unicode.IsSpace)

	return func(yield func(string) bool) {
		if value == "" {
			yield("")

			return
		}

		for line := range strings.Lines(value) {
			line = strings.TrimRightFunc(line, unicode.IsSpace)
			if !yield(line) {
				return
			}
		}
	}
}
