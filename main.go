// Command shebang runs scripts using command-line interfaces defined in embedded KDL.
package main

import (
	"errors"
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

var errUnsupportedShell = errors.New("unsupported shell")

func main() {
	debugMode := env.IsDebug()

	log.Configure(debugMode)

	if wantsReadme(os.Args[1:]) {
		renderReadme()
		os.Exit(0)
	}

	scriptPath, err := getScript()
	if err != nil {
		log.Dief("cannot detect a script %s: %s", os.Args[1], err.Error())
	}

	log.PrintVal("Script", scriptPath)

	conf, err := parseConfig(scriptPath)
	if err != nil {
		log.Dief("cannot read a config for %s: %s", scriptPath, err)
	}

	execFunc := runExecve
	if debugMode {
		execFunc = runDebug
	}

	cmd := cli.NewCommand(scriptPath, execFunc)
	if err := conf.Configure(cmd); err != nil {
		log.Dief("cannot configure command: %s", err)
	}

	cmd.Cmd.SetArgs(os.Args[2:])

	if _, ok := os.LookupEnv(env.Var("COMPLETION")); ok {
		if err := runCompletion(&cmd.Cmd); err != nil {
			log.Dief("cannot generate shell completions: %s", err)
		}
	} else {
		if err := cmd.Cmd.Execute(); err != nil {
			log.Dief("cannot execute command: %s", err)
		}
	}
}

//nolint:ireturn // Preserve the version-independent configuration returned by config.Parse.
func parseConfig(path string) (config.Config, error) {
	file, err := os.Open(path) //nolint:gosec // G304: Read the script selected by the user.
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	return config.Parse(file)
}

func getScript() (string, error) {
	scriptPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		return "", err
	}

	return exec.LookPath(scriptPath)
}

func runExecve(_ *cobra.Command, args []string) error {
	// G204: Execute the interpreter resolved from the script configuration.
	return syscall.Exec(args[0], args, os.Environ()) //nolint:gosec
}

func runDebug(cmd *cobra.Command, args []string) error {
	cmd.Println("Argv:", args)
	cmd.Println("Environment:")

	for _, v := range os.Environ() {
		k, _, _ := strings.Cut(v, "=")
		if strings.HasPrefix(k, env.Prefix) {
			cmd.Println(v)
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
	case "pwsh":
		return cmd.GenPowerShellCompletion(cmd.OutOrStdout())
	}

	return fmt.Errorf("%w %s", errUnsupportedShell, shell)
}
