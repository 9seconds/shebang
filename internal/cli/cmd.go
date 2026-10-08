// Package cli builds script commands and exports parsed options and flags.
package cli

import (
	"github.com/spf13/cobra"
)

// Command combines CLI parsing with script execution.
type Command struct {
	Cmd        cobra.Command
	ScriptName string
	Argv       []string
	Options    []Option
	Flags      []Flag
}

// NewCommand creates a script command that invokes execute with the interpreter,
// script path, and positional arguments.
func NewCommand(scriptName string, execute func(*cobra.Command, []string) error) *Command {
	cmd := &Command{
		ScriptName: scriptName,
		Cmd: cobra.Command{
			DisableAutoGenTag:     true,
			DisableFlagsInUseLine: true,
			CompletionOptions: cobra.CompletionOptions{
				DisableDefaultCmd: true,
				DisableNoDescFlag: true,
				HiddenDefaultCmd:  true,
			},
		},
	}
	cmd.Cmd.Flags().SortFlags = true
	cmd.Cmd.CompletionOptions.SetDefaultShellCompDirective(
		cobra.ShellCompDirectiveNoFileComp,
	)

	cmd.Cmd.PreRun = func(_ *cobra.Command, _ []string) {
		for _, opt := range cmd.Options {
			opt.SetEnv()
		}

		for _, flag := range cmd.Flags {
			flag.SetEnv()
		}
	}
	cmd.Cmd.RunE = func(command *cobra.Command, args []string) error {
		toExecute := make([]string, 0, len(cmd.Argv)+1+len(args))
		toExecute = append(toExecute, cmd.Argv...)
		toExecute = append(toExecute, cmd.ScriptName)
		toExecute = append(toExecute, args...)

		return execute(command, toExecute)
	}

	return cmd
}
