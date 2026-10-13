package cli_test

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type CommandTestSuite struct {
	suite.Suite
}

func (suite *CommandTestSuite) TestNewCommand() {
	// Act
	cmd := cli.NewCommand("/scripts/script", nil)

	// Assert
	suite.Equal("/scripts/script", cmd.ScriptName)
	suite.True(cmd.Cmd.DisableAutoGenTag)
	suite.True(cmd.Cmd.DisableFlagsInUseLine)
	suite.True(cmd.Cmd.Flags().SortFlags)
	suite.True(cmd.Cmd.CompletionOptions.DisableDefaultCmd)
	suite.True(cmd.Cmd.CompletionOptions.DisableNoDescFlag)
	suite.True(cmd.Cmd.CompletionOptions.HiddenDefaultCmd)
	suite.Require().NotNil(cmd.Cmd.PreRun)
	suite.Require().NotNil(cmd.Cmd.RunE)
}

func (suite *CommandTestSuite) TestExecute() {
	wantError := errors.New("execution failed")
	for _, test := range []struct {
		name string
		argv []string
		args []string
		want []string
		err  error
	}{
		{
			name: "script only",
			argv: []string{"runner"},
			args: []string{},
			want: []string{
				"runner",
				"/scripts/script",
			},
		},
		{
			name: "runner and script arguments",
			argv: []string{
				"runner",
				"-eu",
			},
			args: []string{
				"привет",
				"dest",
			},
			want: []string{
				"runner",
				"-eu",
				"/scripts/script",
				"привет",
				"dest",
			},
		},
		{
			name: "dash arguments after separator",
			argv: []string{"runner"},
			args: []string{
				"--",
				"-value",
			},
			want: []string{
				"runner",
				"/scripts/script",
				"-value",
			},
		},
		{
			name: "completion remains a script argument",
			argv: []string{"runner"},
			args: []string{
				"completion",
				"bash",
			},
			want: []string{
				"runner",
				"/scripts/script",
				"completion",
				"bash",
			},
		},
		{
			name: "execution error propagated",
			argv: []string{"runner"},
			args: []string{},
			want: []string{
				"runner",
				"/scripts/script",
			},
			err: wantError,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			suite.T().Setenv("SHEBANG_TEST_INHERITED", "привет")

			calls := 0

			var received []string

			cmd := cli.NewCommand("/scripts/script", func(_ *cobra.Command, argv []string) error {
				calls++
				received = argv

				return test.err
			})
			cmd.Argv = test.argv
			cmd.Cmd.SetOut(io.Discard)
			cmd.Cmd.SetErr(io.Discard)

			cmd.Cmd.SetArgs(test.args)
			suite.Zero(calls)

			// Act
			err := cmd.Cmd.Execute()

			// Assert
			if test.err == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, test.err)
			}

			suite.Equal(1, calls)
			suite.Equal(test.want, received)
		})
	}
}

func (suite *CommandTestSuite) TestExecuteExportsEnvironment() {
	// Arrange
	for _, key := range []string{
		"SHEBANG_OL_OUTPUT",
		"SHEBANG_OS_O",
		"SHEBANG_FL_VERBOSE",
		"SHEBANG_FS_V",
	} {
		suite.T().Setenv(key, "old")
	}

	validator, err := values.NewStr(nil, nil)
	suite.Require().NoError(err)

	called := false

	var received, environment []string

	cmd := cli.NewCommand("script", func(_ *cobra.Command, argv []string) error {
		called = true
		received = argv
		environment = os.Environ()

		return nil
	})
	cmd.Argv = []string{"runner"}
	cmd.Options = []cli.Option{
		{
			Long:      "output",
			Short:     "o",
			Validator: validator,
		},
	}
	cmd.Flags = []cli.Flag{
		{
			Long:  "verbose",
			Short: "v",
		},
	}
	cmd.Cmd.Flags().VarP(&cmd.Options[0], "output", "o", "Output")
	cmd.Cmd.Flags().BoolVarP(&cmd.Flags[0].Value, "verbose", "v", false, "Verbose")
	cmd.Cmd.SetArgs([]string{
		"-o",
		"привет",
		"-v",
		"source",
	})
	suite.False(called)
	suite.Nil(cmd.Options[0].Value)
	suite.False(cmd.Flags[0].Value)

	for _, key := range []string{
		"SHEBANG_OL_OUTPUT",
		"SHEBANG_OS_O",
		"SHEBANG_FL_VERBOSE",
		"SHEBANG_FS_V",
	} {
		suite.Equal("old", os.Getenv(key))
	}

	// Act
	err = cmd.Cmd.Execute()

	// Assert
	suite.Require().NoError(err)
	suite.True(called)
	suite.Equal([]string{
		"runner",
		"script",
		"source",
	}, received)

	for _, entry := range []string{
		"SHEBANG_OL_OUTPUT=привет",
		"SHEBANG_OS_O=привет",
		"SHEBANG_FL_VERBOSE=true",
		"SHEBANG_FS_V=true",
	} {
		suite.Contains(environment, entry)
	}
}

func (suite *CommandTestSuite) TestRejectedArgumentsDoNotExecute() {
	want := errors.New("arguments rejected")

	for _, test := range []struct {
		name     string
		args     []string
		validate bool
	}{
		{
			name: "unknown flag",
			args: []string{"--unknown"},
		},
		{
			name:     "positional validation fails",
			args:     []string{"value"},
			validate: true,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			calls := 0
			cmd := cli.NewCommand("script", func(*cobra.Command, []string) error {
				calls++

				return nil
			})
			cmd.Argv = []string{"runner"}
			cmd.Cmd.SetOut(io.Discard)
			cmd.Cmd.SetErr(io.Discard)

			if test.validate {
				cmd.Cmd.Args = func(*cobra.Command, []string) error {
					return want
				}
			}

			cmd.Cmd.SetArgs(test.args)

			// Act
			err := cmd.Cmd.Execute()

			// Assert
			suite.Require().Error(err)

			if test.validate {
				suite.Require().ErrorIs(err, want)
			} else {
				suite.Contains(strings.ToLower(err.Error()), "unknown flag")
			}

			suite.Zero(calls)
		})
	}
}

//nolint:paralleltest // The suite mutates process environment variables.
func TestCommand(t *testing.T) {
	suite.Run(t, &CommandTestSuite{})
}
