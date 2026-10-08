package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/9seconds/shebang/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		die("usage: shebang <script> <arg>...")
	}

	scriptName, err := getScript()
	if err != nil {
		die("cannot execute %s: %s", os.Args[1], err)
	}

	conf, err := parseConfig(scriptName)
	if err != nil {
		die("cannot read config: %s", err)
	}

	fmt.Println(conf)
}

func die(format string, arg ...any) {
	fmt.Fprintf(os.Stderr, format, arg...)
	fmt.Fprint(os.Stderr, "\n")
	os.Exit(1)
}

func parseConfig(path string) (config.Config, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer fp.Close()

	return config.Parse(fp)
}

func getScript() (string, error) {
	scriptName, err := filepath.Abs(os.Args[1])
	if err != nil {
		die("cannot execute %s: %s", os.Args[1], err)
	}

	return exec.LookPath(scriptName)
}
