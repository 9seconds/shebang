package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/9seconds/shebang/internal/config"
	"github.com/9seconds/shebang/internal/env"
	"github.com/9seconds/shebang/internal/log"
	"github.com/spf13/cobra"
)

func main() {
	debugMode := env.IsDebug()

	log.Configure(debugMode)

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


	execFunc := runExecve
	if debugMode {
		execFunc = runDebug
	}

	cmd := cli.NewCommand(scriptPath, execFunc)
	if err := conf.Configure(cmd); err != nil {
		log.Die("cannot configure command: %s", err)
	}

	if _, ok := os.LookupEnv(env.Var("COMPLETION")); ok {
		if err := runCompletion(&cmd.Cmd); err != nil {
			log.Die("cannot generate shell completions: %s", err)
		}
	} else {
		if err := cmd.Execute(os.Args[2:]); err != nil {
			log.Die("cannot execute command: %s", err)
		}
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

func runExecve(args []string) error {
	return syscall.Exec(args[0], args, os.Environ())
}

func runDebug(args []string) error {
	fmt.Println("Argv:", args)
	fmt.Println("Environment:")

	for _, v := range os.Environ() {
		k, _, _ := strings.Cut(v, "=")
		if strings.HasPrefix(k, env.PREFIX) {
			fmt.Println(v)
		}
	}

	return nil
}

func runCompletion(cmd *cobra.Command) error {
	shell := os.Getenv(env.Var("COMPLETION"))
	if shell == "" || shell == "auto" {
		shell = os.Getenv("SHELL")
	}

	switch filepath.Base(shell) {
	case "bash":
		return cmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
	case "zsh":
		return cmd.GenZshCompletion(cmd.OutOrStdout())
	case "fish":
		return cmd.GenFishCompletion(cmd.OutOrStdout(), true)
	case "power", "powershell":
		return cmd.GenPowerShellCompletion(cmd.OutOrStdout())
	}

	log.Die("unsupported shell %s", shell)

	return nil
}
