package cli

import (
	"github.com/spf13/cobra"
)

type Command struct {
	Cmd cobra.Command
	ScriptName string
	Argv []string
	Options []Option
	Flags []Flag
}

func (c *Command) Execute(args []string) error {
	c.Cmd.SetArgs(args)
	return c.Cmd.Execute()
}

func NewCommand(scriptName string, execute func ([]string) error) *Command {
	cmd := &Command{
		ScriptName: scriptName,
		Cmd: cobra.Command{
			DisableAutoGenTag: true,
			DisableFlagsInUseLine: true,
			CompletionOptions: cobra.CompletionOptions{
				DisableDefaultCmd: true,
				DisableNoDescFlag: true,
				HiddenDefaultCmd: true,
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
	cmd.Cmd.RunE = func(_ *cobra.Command, args []string) error {
		toExecute := append(cmd.Argv, cmd.ScriptName)
		toExecute = append(toExecute, args...)

		return execute(toExecute)
	}

	return cmd
}
