package v1_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/cli"
	v1 "github.com/9seconds/shebang/internal/config/v1"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
	runner string
}

func (suite *ConfigTestSuite) SetupTest() {
	runner, err := os.Executable()

	suite.Require().NoError(err)
	suite.runner = runner
}

func (suite *ConfigTestSuite) TestHelpText() {
	for _, test := range []struct {
		name string
		conf v1.Config
		want string
	}{
		{
			name: "empty description",
		},
		{
			name: "description without arguments preserves whitespace",
			conf: v1.Config{
				Description: "  привет\n  ",
				Example:     "script source dest",
			},
			want: "  привет\n  ",
		},
		{
			name: "positional descriptions in declaration order",
			conf: v1.Config{
				Description: "A script",
				Example:     "script source item dest",
				FirstArgs: []v1.Arg{
					{
						Name:        "source",
						Description: "Source file",
					},
				},
				VarArgs: &v1.VarArg{
					Arg: v1.Arg{
						Name:        "items",
						Description: "Input items",
					},
				},
				LastArgs: []v1.Arg{
					{
						Name:        "dest",
						Description: "Destination",
					},
				},
			},
			want: "A script\n\nPositional arguments:\n  SOURCE   Source file\n  ITEMS    Input items\n  DEST     Destination",
		},
		{
			name: "unicode alignment uses runes",
			conf: v1.Config{
				FirstArgs: []v1.Arg{
					{
						Name:        "привет",
						Description: "Greeting",
					},
				},
				LastArgs: []v1.Arg{
					{
						Name:        "end",
						Description: "End",
					},
				},
			},
			want: "Positional arguments:\n  ПРИВЕТ   Greeting\n  END      End",
		},
	} {
		suite.Run(test.name, func() {
			conf := test.conf
			conf.Argv = []string{suite.runner}
			cmd := cli.NewCommand("/scripts/script", nil)

			suite.Require().NoError(conf.Configure(cmd))
			suite.Equal(test.want, cmd.Cmd.Long)
			suite.Equal(test.conf.Example, cmd.Cmd.Example)
		})
	}
}

func (suite *ConfigTestSuite) TestUse() {
	for _, test := range []struct {
		name string
		doc  string
		want string
	}{
		{
			name: "no arguments",
			want: "script",
		},
		{
			name: "flag",
			doc:  "flag \"verbose\"\n",
			want: "script [flags]",
		},
		{
			name: "option and positional arguments",
			doc:  "option \"output\"\narg \"source\"\narg \"dest\"\n",
			want: "script [flags] [--] SOURCE DEST",
		},
		{
			name: "unbounded vararg",
			doc:  "vararg \"items\"\n",
			want: "script [--] [ITEMS1 ...]",
		},
		{
			name: "zero minimum",
			doc:  "vararg \"items\" { min-count 0; }\n",
			want: "script [--] [ITEMS1 ...]",
		},
		{
			name: "one required item",
			doc:  "vararg \"items\" { min-count 1; }\n",
			want: "script [--] ITEMS1 [ITEMS2 ...]",
		},
		{
			name: "two required items",
			doc:  "vararg \"items\" { min-count 2; }\n",
			want: "script [--] ITEMS1 ITEMS2 [ITEMS3 ...]",
		},
		{
			name: "many required items",
			doc:  "vararg \"items\" { min-count 4; }\n",
			want: "script [--] ITEMS1 ... ITEMS4 [ITEMS5 ...]",
		},
		{
			name: "zero maximum",
			doc:  "vararg \"items\" { max-count 0; }\n",
			want: "script",
		},
		{
			name: "equal bounds",
			doc:  "vararg \"items\" {\nmin-count 2\nmax-count 2\n}\n",
			want: "script [--] ITEMS1 ITEMS2",
		},
		{
			name: "one optional item",
			doc:  "vararg \"items\" {\nmin-count 1\nmax-count 2\n}\n",
			want: "script [--] ITEMS1 [ITEMS2]",
		},
		{
			name: "two optional items",
			doc:  "vararg \"items\" {\nmin-count 1\nmax-count 3\n}\n",
			want: "script [--] ITEMS1 [ITEMS2 ITEMS3]",
		},
		{
			name: "many optional items with surrounding arguments",
			doc:  "arg \"source\"\nvararg \"items\" { max-count 5; }\narg \"dest\"\n",
			want: "script [--] SOURCE [ITEMS1 ... ITEMS5] DEST",
		},
	} {
		suite.Run(test.name, func() {
			conf, err := v1.Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)

			conf.Argv = []string{suite.runner}
			cmd := cli.NewCommand("/scripts/script", nil)

			suite.Require().NoError(conf.Configure(cmd))
			suite.Equal(test.want, cmd.Cmd.Use)
			suite.Equal(test.want, cmd.Cmd.UseLine())
		})
	}
}

func (suite *ConfigTestSuite) TestInterpreter() {
	for _, test := range []struct {
		name string
		argv []string
	}{
		{
			name: "absolute executable",
			argv: []string{suite.runner},
		},
		{
			name: "executable arguments",
			argv: []string{suite.runner, "-eu", "-o", "pipefail"},
		},
		{
			name: "executable resolved through path",
			argv: []string{filepath.Base(suite.runner), "-x"},
		},
	} {
		suite.Run(test.name, func() {
			suite.T().Setenv("PATH", filepath.Dir(suite.runner))

			conf := &v1.Config{
				Argv: test.argv,
			}
			cmd := cli.NewCommand("script", nil)
			suite.Require().NoError(conf.Configure(cmd))

			want := append([]string{suite.runner}, test.argv[1:]...)
			suite.Equal(want, cmd.Argv)
			suite.Equal(test.argv, conf.Argv)
		})
	}
}

func (suite *ConfigTestSuite) TestOptionsAndFlags() {
	conf, err := v1.Parse(strings.NewReader(`option "output" {
		short "o"
		description "Output value"
		value "str" { re "^привет$"; }
	}
	option "plain"
	flag "verbose" {
		short "v"
		description "Verbose output"
	}
	flag "quiet"
	`))
	suite.Require().NoError(err)

	conf.Argv = []string{suite.runner}
	cmd := cli.NewCommand("script", nil)
	suite.Require().NoError(conf.Configure(cmd))
	suite.Require().Len(cmd.Options, 2)
	suite.Require().Len(cmd.Flags, 2)

	for _, test := range []struct {
		name        string
		short       string
		description string
		valueType   string
	}{
		{
			name:        "output",
			short:       "o",
			description: "Output value",
			valueType:   "str",
		},
		{
			name: "plain",
		},
		{
			name:        "verbose",
			short:       "v",
			description: "Verbose output",
			valueType:   "bool",
		},
		{
			name:      "quiet",
			valueType: "bool",
		},
	} {
		suite.Run(test.name, func() {
			flag := cmd.Cmd.Flags().Lookup(test.name)

			suite.Require().NotNil(flag)
			suite.Equal(test.short, flag.Shorthand)
			suite.Equal(test.description, flag.Usage)
			suite.Equal(test.valueType, flag.Value.Type())
		})
	}

	suite.Equal("output", cmd.Options[0].Long)
	suite.Equal("o", cmd.Options[0].Short)
	suite.Equal("str", cmd.Options[0].ValueType)
	suite.Equal("plain", cmd.Options[1].Long)
	suite.Nil(cmd.Options[0].Value)
	suite.Nil(cmd.Options[1].Value)
	suite.Equal([]cli.Flag{
		{
			Long:  "verbose",
			Short: "v",
		},
		{
			Long: "quiet",
		},
	}, cmd.Flags)
	suite.Error(cmd.Cmd.Flags().Set("output", "invalid"))
	suite.Nil(cmd.Options[0].Value)
	suite.Require().NoError(cmd.Cmd.ParseFlags([]string{"-o", "привет", "--plain", "text", "-v", "--quiet"}))
	suite.Equal(new("привет"), cmd.Options[0].Value)
	suite.Equal(new("text"), cmd.Options[1].Value)
	suite.True(cmd.Flags[0].Value)
	suite.True(cmd.Flags[1].Value)

	for _, name := range []string{"output", "plain"} {
		suite.Run(name+" completion", func() {
			complete, ok := cmd.Cmd.GetFlagCompletionFunc(name)
			suite.Require().True(ok)
			completions, directive := complete(&cmd.Cmd, nil, "привет")
			suite.Nil(completions)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func (suite *ConfigTestSuite) TestPositionalCallbacks() {
	for _, test := range []struct {
		name    string
		args    []string
		message string
	}{
		{
			name: "valid argument",
			args: []string{"привет"},
		},
		{
			name:    "missing argument",
			message: "there must be at least 1 arguments, got 0",
		},
		{
			name:    "invalid argument",
			args:    []string{"invalid"},
			message: "invalid argument WORD:",
		},
		{
			name:    "extra argument",
			args:    []string{"привет", "extra"},
			message: "expected 0 variadic arguments, got 1",
		},
	} {
		suite.Run(test.name, func() {
			conf, err := v1.Parse(strings.NewReader("arg \"word\" { value \"str\" { re \"^привет$\"; }; }\n"))
			suite.Require().NoError(err)

			conf.Argv = []string{suite.runner}
			cmd := cli.NewCommand("script", nil)
			suite.Require().NoError(conf.Configure(cmd))
			suite.Require().NotNil(cmd.Cmd.Args)
			suite.Require().NotNil(cmd.Cmd.ValidArgsFunction)

			err = cmd.Cmd.Args(&cmd.Cmd, test.args)

			if test.message == "" {
				suite.NoError(err)
			} else {
				suite.ErrorContains(err, test.message)
			}

			completions, directive := cmd.Cmd.ValidArgsFunction(&cmd.Cmd, test.args, "привет")
			suite.Nil(completions)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func (suite *ConfigTestSuite) TestConfigureErrors() {
	for _, test := range []struct {
		name    string
		doc     string
		runner  string
		message string
		want    error
	}{
		{
			name:    "unknown interpreter",
			runner:  "shebang-test-missing-runner",
			message: "unknown runner shebang-test-missing-runner:",
			want:    exec.ErrNotFound,
		},
		{
			name:    "invalid option type",
			doc:     "option \"output\" { value \"unknown\"; }\n",
			message: "incorrect option output:",
			want:    values.ErrUnknownValueType,
		},
		{
			name:    "invalid option property",
			doc:     "option \"output\" { value \"str\" { unknown 1; }; }\n",
			message: "incorrect option output:",
			want:    values.ErrUnknownProperty,
		},
		{
			name:    "invalid positional type",
			doc:     "arg \"source\" { value \"unknown\"; }\n",
			message: "invalid argument SOURCE:",
			want:    values.ErrUnknownValueType,
		},
	} {
		suite.Run(test.name, func() {
			conf, err := v1.Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)
			conf.Argv = []string{suite.runner}
			if test.runner != "" {
				suite.T().Setenv("PATH", suite.T().TempDir())
				conf.Argv = []string{test.runner}
			}
			cmd := cli.NewCommand("script", nil)
			err = conf.Configure(cmd)
			suite.Require().Error(err)
			suite.ErrorContains(err, test.message)
			suite.ErrorIs(err, test.want)
		})
	}
}

func TestConfig(t *testing.T) {
	suite.Run(t, &ConfigTestSuite{})
}
