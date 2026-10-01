package cli

import (
	"fmt"
	"errors"

	"github.com/spf13/cobra"
)

var ErrStop = errors.New("stop executing")

type Command struct {
	cobra.Command

	ExecuteAs []string
	Options []*Option
}

func (c *Command) Process(args []string) ([]string, error) {
	c.SetArgs(args)

	posArgs := []string(nil)
	executed := false

	c.PreRunE = func( _*cobra.Command, _ []string) error {
		for _, flag := range c.Options {
			if err := flag.Validate(); err != nil {
				return fmt.Errorf("invalid flag %s: %w", flag.name, err)
			}
		}
		return nil
	}

	c.Run = func(_ *cobra.Command, args []string) {
		posArgs = args
		executed = true
	}

	c.PostRunE = func(_ *cobra.Command, _ []string) error {
		for _, flag := range c.Options {
			if err := flag.SetEnv(); err != nil {
				return fmt.Errorf("cannot set environment variable %s: %w", Env(flag.name), err)
			}
		}
		return nil
	}

	err := c.Execute()

	if err == nil && !executed {
		err = ErrStop
	}

	return posArgs, err
}

func NewCommand() *Command {
	return &Command{
		DisableAutoGenTag: true,
	}
}
