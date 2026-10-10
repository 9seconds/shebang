package cli_test

import (
	"os"
	"testing"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/9seconds/shebang/internal/values"
	"github.com/stretchr/testify/suite"
)

type OptionTestSuite struct {
	suite.Suite
}

func (suite *OptionTestSuite) TestSet() {
	for _, test := range []struct {
		name       string
		initial    *string
		value      string
		properties map[string][]any
		want       *string
		message    string
	}{
		{
			name:  "set value",
			value: "привет",
			want:  new("привет"),
		},
		{
			name: "empty value",
			want: new(""),
		},
		{
			name:    "replace value",
			initial: new("old"),
			value:   "new",
			want:    new("new"),
		},
		{
			name:  "valid constrained value",
			value: "привет",
			properties: map[string][]any{
				"re": {"^привет$"},
			},
			want: new("привет"),
		},
		{
			name:  "invalid value leaves unset",
			value: "bad",
			properties: map[string][]any{
				"re": {"^привет$"},
			},
			message: "bad does not match ^привет$",
		},
		{
			name:    "invalid value preserves previous value",
			initial: new("привет"),
			value:   "bad",
			properties: map[string][]any{
				"re": {"^привет$"},
			},
			want:    new("привет"),
			message: "bad does not match ^привет$",
		},
	} {
		suite.Run(test.name, func() {
			validator, err := values.NewStr(test.properties)
			suite.Require().NoError(err)

			option := cli.Option{
				Value:     test.initial,
				Validator: validator,
			}

			err = option.Set(test.value)
			if test.message == "" {
				suite.Require().NoError(err)
			} else {
				suite.Require().EqualError(err, test.message)
			}

			suite.Equal(test.want, option.Value)
		})
	}
}

func (suite *OptionTestSuite) TestMetadata() {
	for _, test := range []struct {
		name      string
		short     string
		valueType string
		want      string
	}{
		{
			name: "without short name",
			want: "output  (value=str(subvalidators=))",
		},
		{
			name:      "with short name",
			short:     "o",
			valueType: "str",
			want:      "output o (value=str(subvalidators=))",
		},
	} {
		suite.Run(test.name, func() {
			validator, err := values.NewStr(nil)
			suite.Require().NoError(err)

			option := cli.Option{
				Long:      "output",
				Short:     test.short,
				ValueType: test.valueType,
				Value:     new("привет"),
				Validator: validator,
			}
			suite.Empty(option.String())
			suite.Equal(test.valueType, option.Type())
			suite.Equal(test.want, option.AsString())
		})
	}
}

func (suite *OptionTestSuite) TestSetEnv() {
	for _, test := range []struct {
		name      string
		value     *string
		short     string
		wantLong  string
		wantShort string
	}{
		{
			name:      "unset preserves environment",
			short:     "o",
			wantLong:  "old",
			wantShort: "old",
		},
		{
			name:      "set both names",
			value:     new("привет"),
			short:     "o",
			wantLong:  "привет",
			wantShort: "привет",
		},
		{
			name:      "set long name only",
			value:     new("привет"),
			wantLong:  "привет",
			wantShort: "old",
		},
		{
			name:  "empty value exported",
			value: new(""),
			short: "o",
		},
	} {
		suite.Run(test.name, func() {
			suite.T().Setenv("SHEBANG_OL_OUTPUT", "old")
			suite.T().Setenv("SHEBANG_OS_O", "old")

			option := cli.Option{
				Long:  "output",
				Short: test.short,
				Value: test.value,
			}
			option.SetEnv()
			suite.Equal(test.wantLong, os.Getenv("SHEBANG_OL_OUTPUT"))
			suite.Equal(test.wantShort, os.Getenv("SHEBANG_OS_O"))
		})
	}
}

//nolint:paralleltest // The suite mutates process environment variables.
func TestOption(t *testing.T) {
	suite.Run(t, &OptionTestSuite{})
}
