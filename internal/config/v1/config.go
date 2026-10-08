package v1

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/9seconds/shebang/internal/log"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
)

type WithValue struct {
	Type string
	Properties map[string][]any
}

type Flag struct {
	Name string
	Description string
	Short string
}

type Option struct {
	Flag
	WithValue
}

type Arg struct {
	WithValue

	Name string
	Description string
}

type VarArg struct {
	Arg

	MinCount *int64
	MaxCount *int64
}

type Config struct {
	Description string
	Example string
	Argv []string
	Flags []Flag
	Options []Option
	FirstArgs []Arg
	LastArgs []Arg
	VarArgs *VarArg

	seenLongNames map[string]bool
	seenShortNames map[string]bool
}

func (c *Config) Configure(cmd *cli.Command) error {
	c.configureExample(cmd)
	log.PrintVal("Example", cmd.Cmd.Example)

	c.configureDescription(cmd)
	log.PrintVal("Description", cmd.Cmd.Long)

	if err := c.configureArgv(cmd); err != nil {
		return err
	}
	log.PrintVal("Argv", fmt.Sprint(cmd.Argv))

	if err := c.configureUse(cmd); err != nil {
		return err
	}
	log.PrintVal("Useline", cmd.Cmd.UseLine())

	if err := c.configureOptions(cmd); err != nil {
		return err
	}

	if err := c.configurePositionalArgs(cmd); err != nil {
		return err
	}

	c.configureFlags(cmd)

	return nil
}

func (c *Config) configureExample(cmd *cli.Command) {
	cmd.Cmd.Example = c.Example
}

func (c *Config) configureDescription(cmd *cli.Command) {
	description := []string(nil)

	fmtLen := 0
	for _, arg := range c.FirstArgs {
		fmtLen = max(fmtLen, utf8.RuneCountInString(arg.Name))
	}
	for _, arg := range c.LastArgs {
		fmtLen = max(fmtLen, utf8.RuneCountInString(arg.Name))
	}
	if c.VarArgs != nil {
		fmtLen = max(fmtLen, utf8.RuneCountInString(c.VarArgs.Name))
	}

	fmtPositionalArgs := fmt.Sprintf("  %%-%ds %%s", fmtLen + 2)

	for _, arg := range c.FirstArgs {
		description = append(
			description,
			fmt.Sprintf(fmtPositionalArgs, strings.ToUpper(arg.Name), arg.Description),
		)
	}

	if c.VarArgs != nil {
		description = append(
			description,
			fmt.Sprintf(
				fmtPositionalArgs,
				strings.ToUpper(c.VarArgs.Name),
				c.VarArgs.Description,
			),
		)
	}

	for _, arg := range c.LastArgs {
		description = append(
			description,
			fmt.Sprintf(fmtPositionalArgs, strings.ToUpper(arg.Name), arg.Description),
		)
	}

	cmd.Cmd.Long = c.Description

	if len(description) > 0 {
		description = append(
			[]string{c.Description, "", "Positional arguments:"},
			description...,
		)
		cmd.Cmd.Long = strings.TrimSpace(strings.Join(description, "\n"))
	}
}

func (c *Config) configureArgv(cmd *cli.Command) error {
	prog, err := exec.LookPath(c.Argv[0])
	if err != nil {
		return fmt.Errorf("unknown runner %s: %w", c.Argv[0], err)
	}

	cmd.Argv = append([]string{prog}, c.Argv[1:]...)

	return nil
}

func (c *Config) configureUse(cmd *cli.Command) error {
	chunks := []string{filepath.Base(cmd.ScriptName)}
	args := []string(nil)

	if len(c.Options) > 0 || len(c.Flags) > 0 {
		chunks = append(chunks, "[flags]")
	}

	for _, arg := range c.FirstArgs {
		args = append(args, strings.ToUpper(arg.Name))
	}

	if c.VarArgs != nil {
		name := strings.ToUpper(c.VarArgs.Name)

		minCount := 0
		if c.VarArgs.MinCount != nil {
			minCount = int(*c.VarArgs.MinCount)
			switch minCount {
			case 0:
			case 1:
				args = append(args, fmt.Sprintf("%s1", name))
			case 2:
				args = append(args, fmt.Sprintf("%s1", name))
				args = append(args, fmt.Sprintf("%s2", name))
			default:
				args = append(args, fmt.Sprintf("%s1", name))
				args = append(args, "...")
				args = append(args, fmt.Sprintf("%s%d", name, minCount))
			}
		}

		if c.VarArgs.MaxCount == nil {
			args = append(args, fmt.Sprintf("[%s%d", name, minCount + 1))
			args = append(args, "...]")
		} else {
			maxCount := int(*c.VarArgs.MaxCount)
			switch maxCount - minCount {
			case 0:
			case 1:
				args = append(args, fmt.Sprintf("[%s%d]", name, minCount + 1))
			case 2:
				args = append(args, fmt.Sprintf("[%s%d", name, minCount + 1))
				args = append(args, fmt.Sprintf("%s%d]", name, minCount + 2))
			default:
				args = append(args, fmt.Sprintf("[%s%d", name, minCount + 1))
				args = append(args, "...")
				args = append(args, fmt.Sprintf("%s%d]", name, maxCount))
			}
		}
	}

	for _, arg := range c.LastArgs {
		args = append(args, strings.ToUpper(arg.Name))
	}

	if len(args) > 0 {
		chunks = append(chunks, "[--]")
	}

	chunks = append(chunks, args...)

	cmd.Cmd.Use = strings.Join(chunks, " ")

	return nil
}

func (c *Config) configureOptions(cmd *cli.Command) error {
	cmd.Options = make([]cli.Option, len(c.Options))
	flagSet := cmd.Cmd.Flags()

	for idx, opt := range c.Options {
		value, err := values.New(opt.Type, opt.Properties)
		if err != nil {
			return fmt.Errorf("incorrect option %s: %w", opt.Name, err)
		}

		cmd.Options[idx].Long = opt.Name
		cmd.Options[idx].Short = opt.Short
		cmd.Options[idx].ValueType = opt.Type
		cmd.Options[idx].Validator = value

		flagSet.VarP(&cmd.Options[idx], opt.Name, opt.Short, opt.Description)
		err = cmd.Cmd.RegisterFlagCompletionFunc(
			opt.Name,
			func(
				_ *cobra.Command,
				_ []string,
				toComplete string,
			) ([]cobra.Completion, cobra.ShellCompDirective) {
				return value.Complete(toComplete)
			},
		)

		if err != nil {
			return err
		}

		log.PrintVal("Option", cmd.Options[idx].AsString())
	}

	return nil
}

func (c *Config) configureFlags(cmd *cli.Command) {
	cmd.Flags = make([]cli.Flag, len(c.Flags))
	flagSet := cmd.Cmd.Flags()

	for idx, fl := range c.Flags {
		cmd.Flags[idx].Long = fl.Name
		cmd.Flags[idx].Short = fl.Short
		flagSet.BoolVarP(&cmd.Flags[idx].Value, fl.Name, fl.Short, false, fl.Description)

		log.PrintVal("Flag", fmt.Sprintf("%s %s", fl.Name, fl.Short))
	}
}

func (c *Config) configurePositionalArgs(cmd *cli.Command) error {
	argVals, err := newArgValidators(c)
	if err != nil {
		return err
	}

	cmd.Cmd.Args = func(_ *cobra.Command, args []string) error {
		return argVals.Validate(args)
	}

	cmd.Cmd.ValidArgsFunction = func(
		_ *cobra.Command,
		args []string,
		toComplete string,
	) ([]cobra.Completion, cobra.ShellCompDirective) {
		return argVals.Complete(args, toComplete)
	}

	return nil
}
