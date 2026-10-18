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
		repr    string
		valid   bool
	}{
		{
			name: "exact match",
			choices: []any{
				"dev",
				"prod",
			},
			input: "prod",
			repr:  "enum(choices:dev,prod)",
			valid: true,
		},
		{
			name:    "unicode match",
			choices: []any{"привет"},
			input:   "привет",
			repr:    "enum(choices:привет)",
			valid:   true,
		},
		{
			name: "duplicate choices",
			choices: []any{
				"dev",
				"dev",
			},
			input: "dev",
			repr:  "enum(choices:dev)",
			valid: true,
		},
		{
			name:    "explicit empty choice",
			choices: []any{""},
			repr:    "enum(choices)",
			valid:   true,
		},
		{
			name:    "case sensitive",
			choices: []any{"prod"},
			input:   "Prod",
			repr:    "enum(choices:prod)",
		},
		{
			name:    "prefix is not a valid choice",
			choices: []any{"prod"},
			input:   "pro",
			repr:    "enum(choices:prod)",
		},
		{
			name:    "whitespace is not trimmed",
			choices: []any{"prod"},
			input:   " prod ",
			repr:    "enum(choices:prod)",
		},
		{
			name:    "empty choice list",
			choices: []any{},
			input:   "dev",
			repr:    "enum(choices)",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.New("enum", nil, map[string][]any{
				"choices": test.choices,
			})
			suite.Require().NoError(err)
			suite.Equal(test.repr, value.String())

			// Act
			err = value.Validate(test.input)

			// Assert
			if test.valid {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, values.ErrEnumUnknownChoice)
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
			name: "unknown subvalidator",
			properties: map[string][]any{
				"unknown": {"dev"},
			},
			want: values.ErrUnknownSubvalidator,
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
				"choices": {
					"dev",
					true,
				},
			},
			typeError: true,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			properties := test.properties

			// Act
			value, err := values.NewEnum(nil, properties)

			// Assert
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
			name: "sorted distinct choices",
			choices: []any{
				"prod",
				"dev",
				"prod",
				"привет",
			},
			want: []cobra.Completion{
				"dev",
				"prod",
				"привет",
			},
		},
		{
			name: "prefix filtering",
			choices: []any{
				"production",
				"preview",
				"dev",
			},
			input: "pr",
			want: []cobra.Completion{
				"preview",
				"production",
			},
		},
		{
			name: "unicode prefix",
			choices: []any{
				"привет",
				"dev",
			},
			input: "при",
			want:  []cobra.Completion{"привет"},
		},
		{
			name: "exact match takes precedence",
			choices: []any{
				"prod",
				"production",
			},
			input: "prod",
			want:  []cobra.Completion{"prod"},
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
			// Arrange
			value, err := values.NewEnum(nil, map[string][]any{
				"choices": test.choices,
			})
			suite.Require().NoError(err)

			// Act
			choices, directive := value.Complete(test.input)

			// Assert
			suite.Equal(test.want, choices)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			for _, choice := range choices {
				suite.Require().NoError(value.Validate(choice))
			}
		})
	}
}

func (suite *EnumTestSuite) TestOmittedChoices() {
	// Arrange
	value, err := values.NewEnum(nil, nil)
	suite.Require().NoError(err)

	// Act
	err = value.Validate("dev")
	choices, directive := value.Complete("")

	// Assert
	suite.Require().ErrorIs(err, values.ErrEnumUnknownChoice)
	suite.Empty(choices)
	suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestEnum(t *testing.T) {
	t.Parallel()

	suite.Run(t, &EnumTestSuite{})
}
