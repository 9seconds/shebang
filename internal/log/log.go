package log

import (
	"io"
	"log"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/9seconds/shebang/internal/cli"
)

func Configure() {
	log.SetOutput(io.Discard)
	log.SetPrefix(">>> ")
	log.SetFlags(0)

	if _, ok := os.LookupEnv(cli.Env("DEBUG")); ok {
		log.SetOutput(os.Stderr)
		_ = os.Unsetenv(cli.Env("DEBUG"))
	}
}

func PrintVal(prefix string, value string) {
	value = strings.TrimSpace(value)
	prefix = prefix + ": "
	spaces := strings.Repeat(" ", len(prefix))
	printedFirst := false
	lines := slices.Collect(strings.Lines(value))

	if len(lines) < 2 {
		log.Println(prefix, value)
		return
	}

	for line := range strings.Lines(value) {
		line  = strings.TrimRightFunc(line, unicode.IsSpace)

		if printedFirst {
			log.Println(spaces, line)
		} else {
			printedFirst = true
			log.Println(prefix, line)
		}
	}
}
