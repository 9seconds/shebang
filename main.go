package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/9seconds/shebang/internal/config"
	"github.com/9seconds/shebang/internal/env"
	"github.com/9seconds/shebang/internal/log"
)

func main() {
	log.Configure(env.Var("debug"))

	if len(os.Args) < 2 {
		log.Die("usage: shebang <script> [arg...]")
	}

	scriptPath, err := getScript()
	if err != nil {
		log.Die("cannot detect a script %s: %s", os.Args[1], err.Error())
	}
	log.PrintVal("Script", scriptPath)

	conf, err := parseConfig(scriptPath)
	if err != nil {
		log.Die("cannot read a config for %s: %s", scriptPath, err)
	}

	cmd := cli.NewCommand(scriptPath, syscall.Exec)
	if err := conf.Configure(cmd); err != nil {
		log.Die("cannot configure command: %s", err)
	}

	if err := cmd.Execute(os.Args[2:]); err != nil {
		log.Die("cannot execute command: %s", err)
	}
}

func parseConfig(path string) (config.Config, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer func() { _ = fp.Close() }()

	return config.Parse(fp)
}

func getScript() (string, error) {
	scriptPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		return "", err
	}

	return exec.LookPath(scriptPath)
}
