package cli

import (
	"fmt"
	"os"

	"github.com/9seconds/shebang/internal/env"
	"github.com/spf13/cobra"
)

type (
	ExecFunc func(string, []string, []string) error
)

type Command struct {
	cobra.Command

	Argv    []string
	Options []*Option
	Flags   []*Flag
}

func (c *Command) Execute(args []string) error {
	c.SetArgs(args)
	return c.Command.Execute()
}

func NewCommand(scriptName string, execute ExecFunc) *Command {
	cmd := &Command{
		DisableAutoGenTag: true,
	}

	cmd.PreRunE = func(_ *cobra.Command, _ []string) error {
		for _, opt := range cmd.Options {
			if err := opt.Validate(); err != nil {
				return fmt.Errorf("invalid option %s: %w", opt.Name, err)
			}
			opt.SetEnv()
		}

		for _, flag := range cmd.Flags {
			if flag.Value {
				env.Set(flag.Name, flag.String())
			}
		}

		return nil
	}

	cmd.RunE = func(_ *cobra.Command, args []string) error {
		toExecute := append(cmd.Argv, scriptName)
		toExecute = append(toExecute, args...)

		return execute(toExecute[0], toExecute, os.Environ())
	}

	return cmd
}
