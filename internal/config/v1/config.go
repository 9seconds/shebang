package v1

import (
	"fmt"
	"os/exec"
	"path/filepath"
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
	valueParams map[string]any
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

	cmd.ExecuteAs = []string{"bash"}
	if len(c.execute) > 0 {
		cmd.ExecuteAs = append([]string(nil), c.execute...)
	}

	if !filepath.IsAbs(cmd.ExecuteAs[0]) {
		path, err := exec.LookPath("env")
		if err != nil {
			return fmt.Errorf("cannot find 'env' in PATH: %w", err)
		}
		cmd.ExecuteAs = []string{path, "-S " + strings.Join(cmd.ExecuteAs, " ")}
	}

	log.PrintVal("Execute As", fmt.Sprint(cmd.ExecuteAs))

	cmd.Options = []*cli.Option{}
	flagSet := cmd.Flags()

	for name, opt := range c.options {
		vld, err := validators.New(opt.valueType, opt.valueParams)
		if err != nil {
			return fmt.Errorf("cannot initialize validator for %s: %w", name, err)
		}

		option := cli.NewOption(name, opt.valueType, opt.minCount, opt.maxCount, vld)
		cmd.Options = append(cmd.Options, option)

		if opt.short == "" {
			flagSet.Var(option, name, opt.description)
		} else {
			flagSet.VarP(option, name, opt.short, opt.description)
		}

		log.PrintVal("Option", option.String())
	}

	return nil
}
