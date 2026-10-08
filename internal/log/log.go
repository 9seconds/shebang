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

func Configure(isDebug bool) {
	if isDebug {
		main.SetOutput(os.Stderr)
	}
}

func Die(format string, arg ...any) {
	format = strings.TrimRightFunc(format, unicode.IsSpace) + "\n"
	err.Fatalf(format, arg...)
}

func Print(format string, arg ...any) {
	PrintVal("", fmt.Sprintf(format, arg...))
}

func PrintVal(reason string, value string) {
	value = strings.TrimSpace(value)

	reason = strings.TrimSpace(reason)
	if reason != "" && !strings.HasSuffix(reason, ":") {
		reason = reason + ": "
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
