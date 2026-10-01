package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/9seconds/shebang/internal/config"
	"github.com/9seconds/shebang/internal/log"
)

func main() {
	if len(os.Args) < 2 {
		die("usage: shebang <script> <arg>...")
	}

	log.Configure()

	scriptName, err := getScript()
	if err != nil {
		die("cannot execute %s: %s", os.Args[1], err)
	}

	conf, err := parseConfig(scriptName)
	if err != nil {
		die("cannot read config: %s", err)
	}

	cmd := cli.NewCommand()
	if err := conf.Configure(cmd); err != nil {
		die("cannot configure CLI: %s", err)
	}

	args, err := cmd.Process(os.Args[2:])
	switch {
	case errors.Is(err, cli.ErrStop):
		return
	case err != nil:
		die("cannot process flags: %s", err)
	}

	execArgs := append(cmd.ExecuteAs, scriptName)
	execArgs = append(execArgs, args...)

	if err := syscall.Exec(execArgs[0], execArgs, os.Environ()); err != nil {
		die("cannot run a command: %s", err)
	}
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
