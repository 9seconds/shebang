package values_test

import (
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type EnumTestSuite struct {
	suite.Suite
}

func (suite *EnumTestSuite) TestValidate() {
	for _, test := range []struct {
		name    string
		choices []any
		input   string
		valid   bool
	}{
		{
			name:    "exact match",
			choices: []any{"dev", "prod"},
			input:   "prod",
			valid:   true,
		},
		{
			name:    "unicode match",
			choices: []any{"привет"},
			input:   "привет",
			valid:   true,
		},
		{
			name:    "duplicate choices",
			choices: []any{"dev", "dev"},
			input:   "dev",
			valid:   true,
		},
		{
			name:    "explicit empty choice",
			choices: []any{""},
			valid:   true,
		},
		{
			name:    "case sensitive",
			choices: []any{"prod"},
			input:   "Prod",
		},
		{
			name:    "prefix is not a valid choice",
			choices: []any{"prod"},
			input:   "pro",
		},
		{
			name:    "whitespace is not trimmed",
			choices: []any{"prod"},
			input:   " prod ",
		},
		{
			name:    "empty choice list",
			choices: []any{},
			input:   "dev",
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.New("enum", map[string][]any{
				"choices": test.choices,
			})
			suite.Require().NoError(err)
			suite.Equal("enum(subvalidators=choices)", value.String())

			err = value.Validate(test.input)
			if test.valid {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, values.ErrUnknownChoice)
			}
		})
	}
}

func (suite *EnumTestSuite) TestProperties() {
	for _, test := range []struct {
		name       string
		properties map[string][]any
		want       error
		typeError  bool
	}{
		{
			name: "unknown property",
			properties: map[string][]any{
				"unknown": {"dev"},
			},
			want: values.ErrUnknownProperty,
		},
		{
			name: "integer choice",
			properties: map[string][]any{
				"choices": {int64(1)},
			},
			typeError: true,
		},
		{
			name: "mixed choice types",
			properties: map[string][]any{
				"choices": {"dev", true},
			},
			typeError: true,
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewEnum(test.properties)
			suite.Require().ErrorContains(err, "cannot add validator")
			suite.Nil(value)

			if test.typeError {
				var typeError *utils.ArgumentTypeError

				suite.Require().ErrorAs(err, &typeError)
				suite.Equal("string", typeError.Expected)
			} else {
				suite.Require().ErrorIs(err, test.want)
			}
		})
	}
}

func (suite *EnumTestSuite) TestCompletion() {
	for _, test := range []struct {
		name    string
		choices []any
		input   string
		want    []cobra.Completion
	}{
		{
			name:    "sorted distinct choices",
			choices: []any{"prod", "dev", "prod", "привет"},
			want:    []cobra.Completion{"dev", "prod", "привет"},
		},
		{
			name:    "prefix filtering",
			choices: []any{"production", "preview", "dev"},
			input:   "pr",
			want:    []cobra.Completion{"preview", "production"},
		},
		{
			name:    "unicode prefix",
			choices: []any{"привет", "dev"},
			input:   "при",
			want:    []cobra.Completion{"привет"},
		},
		{
			name:    "exact match takes precedence",
			choices: []any{"prod", "production"},
			input:   "prod",
			want:    []cobra.Completion{"prod"},
		},
		{
			name:    "case sensitive prefix",
			choices: []any{"prod"},
			input:   "Pr",
			want:    []cobra.Completion{},
		},
		{
			name:    "empty choices",
			choices: []any{},
			want:    []cobra.Completion{},
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewEnum(map[string][]any{
				"choices": test.choices,
			})
			suite.Require().NoError(err)

			choices, directive := value.Complete(test.input)
			suite.Equal(test.want, choices)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			for _, choice := range choices {
				suite.Require().NoError(value.Validate(choice))
			}
		})
	}
}

func (suite *EnumTestSuite) TestOmittedChoices() {
	value, err := values.NewEnum(nil)
	suite.Require().NoError(err)
	suite.Require().ErrorIs(value.Validate("dev"), values.ErrUnknownChoice)

	choices, directive := value.Complete("")
	suite.Empty(choices)
	suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestEnum(t *testing.T) {
	t.Parallel()

	suite.Run(t, &EnumTestSuite{})
}
