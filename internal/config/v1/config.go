package v1

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/9seconds/shebang/internal/log"
	"github.com/9seconds/shebang/internal/validators"
)

type configItem struct {
	description string
	minCount    int
	maxCount    int
	valueType   string
	valueParams map[string][]any
}

type configOption struct {
	configItem
	short string
}

type configArgument struct {
	configItem
}

type Config struct {
	options     map[string]configOption
	argument    configArgument
	execute     []string
	description string
	example     string
}

func (c *Config) Configure(cmd *cli.Command) error {
	cmd.Long = c.description
	log.PrintVal("Description", c.description)

	cmd.Example = c.example
	log.PrintVal("Example", c.example)

	cmd.Argv = []string{"bash"}
	if len(c.execute) > 0 {
		cmd.Argv = slices.Clone(c.execute)
	}
	if !filepath.IsAbs(cmd.Argv[0]) {
		path, err := exec.LookPath("env")
		if err != nil {
			return fmt.Errorf("cannot find 'env' in PATH: %w", err)
		}
		cmd.Argv = []string{path, "-S " + strings.Join(cmd.Argv, " ")}
	}
	log.PrintVal("Argv", strings.Join(cmd.Argv, " "))

	cmd.Options = []*cli.Option{}
	flagSet := cmd.Flags()

	for name, opt := range c.options {
		vld, err := validators.New(opt.valueType, opt.valueParams)
		if err != nil {
			return fmt.Errorf("cannot initialize validator for %s: %w", name, err)
		}

		option := &cli.Option{
			Name: name,
			OptionType: opt.valueType,
			MinCount: opt.minCount,
			MaxCount: opt.maxCount,
			Validator: vld,
		}
		cmd.Options = append(cmd.Options, option)

		if opt.short == "" {
			flagSet.Var(option, name, opt.description)
		} else {
			flagSet.VarP(option, name, opt.short, opt.description)
		}

		log.PrintVal("Option", option.Repr())
	}

	return nil
}
