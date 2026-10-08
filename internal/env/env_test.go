package env_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/env"
	"github.com/stretchr/testify/suite"
)

const (
	processMode  = "SHEBANG_TEST_PROCESS_MODE"
	processDebug = "SHEBANG_TEST_PROCESS_DEBUG"
)

type EnvTestSuite struct {
	suite.Suite
}

func (suite *EnvTestSuite) TestVar() {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "empty name",
			want: "SHEBANG_",
		},
		{
			name:  "uppercase name",
			input: "VALUE",
			want:  "SHEBANG_VALUE",
		},
		{
			name:  "lowercase name",
			input: "value",
			want:  "SHEBANG_VALUE",
		},
		{
			name:  "mixed case name",
			input: "VaLuE",
			want:  "SHEBANG_VALUE",
		},
		{
			name:  "already prefixed",
			input: "SHEBANG_value",
			want:  "SHEBANG_VALUE",
		},
		{
			name:  "prefix only",
			input: "SHEBANG_",
			want:  "SHEBANG_",
		},
		{
			name:  "prefix is case sensitive",
			input: "shebang_value",
			want:  "SHEBANG_SHEBANG_VALUE",
		},
		{
			name:  "prefix must be at the start",
			input: "OTHER_SHEBANG_value",
			want:  "SHEBANG_OTHER_SHEBANG_VALUE",
		},
		{
			name:  "unicode name",
			input: "привет",
			want:  "SHEBANG_ПРИВЕТ",
		},
	} {
		suite.Run(test.name, func() {
			suite.Equal(test.want, env.Var(test.input))
		})
	}
}

func (suite *EnvTestSuite) TestSet() {
	for _, test := range []struct {
		name  string
		input string
		key   string
		value string
	}{
		{
			name:  "unprefixed lowercase name",
			input: "test_value",
			key:   "SHEBANG_TEST_VALUE",
			value: "text",
		},
		{
			name:  "already prefixed name",
			input: "SHEBANG_test_value",
			key:   "SHEBANG_TEST_VALUE",
			value: "text",
		},
		{
			name:  "empty value",
			input: "TEST_VALUE",
			key:   "SHEBANG_TEST_VALUE",
		},
		{
			name:  "value preserves whitespace and punctuation",
			input: "TEST_VALUE",
			key:   "SHEBANG_TEST_VALUE",
			value: "  a=b\nnext line\t",
		},
		{
			name:  "unicode name and value",
			input: "привет",
			key:   "SHEBANG_ПРИВЕТ",
			value: "привет",
		},
		{
			name:  "empty name uses prefix",
			key:   "SHEBANG_",
			value: "text",
		},
	} {
		suite.Run(test.name, func() {
			suite.T().Setenv(test.key, "old")

			env.Set(test.input, test.value)
			value, ok := os.LookupEnv(test.key)

			suite.True(ok)
			suite.Equal(test.value, value)
		})
	}
}

func (suite *EnvTestSuite) TestDebugInitialization() {
	for _, test := range []struct {
		name    string
		value   string
		present bool
		want    string
	}{
		{
			name: "unset",
			want: "false",
		},
		{
			name:    "empty",
			present: true,
			want:    "false",
		},
		{
			name:    "enabled",
			value:   "1",
			present: true,
			want:    "true",
		},
		{
			name:    "false string is nonempty",
			value:   "false",
			present: true,
			want:    "true",
		},
		{
			name:    "zero string is nonempty",
			value:   "0",
			present: true,
			want:    "true",
		},
		{
			name:    "whitespace is nonempty",
			value:   " ",
			present: true,
			want:    "true",
		},
		{
			name:    "unicode value",
			value:   "привет",
			present: true,
			want:    "true",
		},
	} {
		suite.Run(test.name, func() {
			cmd := suite.process("debug")
			cmd.Env = append(cmd.Env, processDebug+"="+test.want)

			if test.present {
				cmd.Env = append(cmd.Env, "SHEBANG_DEBUG="+test.value)
			}

			output, err := cmd.CombinedOutput()
			suite.NoError(err, string(output))
		})
	}
}

func (suite *EnvTestSuite) TestSetErrors() {
	for _, test := range []struct {
		name    string
		mode    string
		message string
	}{
		{
			name:    "equals in name",
			mode:    "equals",
			message: "Cannot set environment variable SHEBANG_BAD=NAME",
		},
		{
			name:    "nul in name",
			mode:    "name nul",
			message: "Cannot set environment variable SHEBANG_BAD\x00NAME",
		},
		{
			name:    "nul in value",
			mode:    "value nul",
			message: "Cannot set environment variable SHEBANG_TEST_VALUE to bad\x00value",
		},
	} {
		suite.Run(test.name, func() {
			output, err := suite.process(test.mode).CombinedOutput()

			var exitError *exec.ExitError

			suite.Require().ErrorAs(err, &exitError)
			suite.Equal(1, exitError.ExitCode())
			suite.Contains(string(output), test.message)
		})
	}
}

func (suite *EnvTestSuite) process(mode string) *exec.Cmd {
	suite.T().Helper()

	executable, err := os.Executable()
	suite.Require().NoError(err)

	cmd := exec.Command(executable, "-test.run=^TestEnvProcess$")

	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if name != "SHEBANG_DEBUG" && name != processMode && name != processDebug {
			cmd.Env = append(cmd.Env, entry)
		}
	}

	cmd.Env = append(cmd.Env, processMode+"="+mode)

	return cmd
}

//nolint:paralleltest // The subprocess helper mutates process environment variables.
func TestEnvProcess(t *testing.T) {
	switch os.Getenv(processMode) {
	case "":
		t.Skip("subprocess helper")

	case "debug":
		checkDebugState(t)

	case "equals":
		env.Set("bad=name", "value")
		t.Fatal("Set returned instead of exiting")
	case "name nul":
		env.Set("bad\x00name", "value")
		t.Fatal("Set returned instead of exiting")
	case "value nul":
		env.Set("TEST_VALUE", "bad\x00value")
		t.Fatal("Set returned instead of exiting")
	default:
		t.Fatal("unknown subprocess mode")
	}
}

func checkDebugState(t *testing.T) {
	t.Helper()

	want := os.Getenv(processDebug) == "true"
	if got := env.IsDebug(); got != want {
		t.Fatalf("IsDebug() = %t, want %t", got, want)
	}

	if _, ok := os.LookupEnv("SHEBANG_DEBUG"); ok {
		t.Fatal("SHEBANG_DEBUG was not removed during initialization")
	}

	t.Setenv("SHEBANG_DEBUG", "")

	if got := env.IsDebug(); got != want {
		t.Fatal("clearing SHEBANG_DEBUG changed the cached debug state")
	}

	t.Setenv("SHEBANG_DEBUG", "1")

	if got := env.IsDebug(); got != want {
		t.Fatal("setting SHEBANG_DEBUG changed the cached debug state")
	}
}

//nolint:paralleltest // The suite mutates process environment variables.
func TestEnv(t *testing.T) {
	suite.Run(t, &EnvTestSuite{})
}
