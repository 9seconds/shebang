package values_test

import (
	"regexp/syntax"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type StrTestSuite struct {
	suite.Suite
}

func (suite *StrTestSuite) TestValidate() {
	for _, test := range []struct {
		name       string
		properties map[string][]any
		input      string
		message    string
	}{
		{
			name: "no constraints accepts empty",
		},
		{
			name:  "no constraints accepts unicode",
			input: "привет",
		},
		{
			name: "minimum below boundary",
			properties: map[string][]any{
				"min-length": {int64(3)},
			},
			input:   "ab",
			message: "minimum length must be at least 3, got 2",
		},
		{
			name: "minimum at boundary",
			properties: map[string][]any{
				"min-length": {int64(6)},
			},
			input: "привет",
		},
		{
			name: "unicode minimum counts runes",
			properties: map[string][]any{
				"min-length": {int64(7)},
			},
			input:   "привет",
			message: "minimum length must be at least 7, got 6",
		},
		{
			name: "zero minimum",
			properties: map[string][]any{
				"min-length": {int64(0)},
			},
		},
		{
			name: "maximum at boundary",
			properties: map[string][]any{
				"max-length": {int64(6)},
			},
			input: "привет",
		},
		{
			name: "maximum above boundary",
			properties: map[string][]any{
				"max-length": {int64(5)},
			},
			input:   "привет",
			message: "minimum length must be at most 5, got 6",
		},
		{
			name: "zero maximum accepts empty",
			properties: map[string][]any{
				"max-length": {int64(0)},
			},
		},
		{
			name: "zero maximum rejects nonempty",
			properties: map[string][]any{
				"max-length": {int64(0)},
			},
			input:   "a",
			message: "minimum length must be at most 0, got 1",
		},
		{
			name: "regex matches unicode",
			properties: map[string][]any{
				"re": {"^привет$"},
			},
			input: "привет",
		},
		{
			name: "regex rejects mismatch",
			properties: map[string][]any{
				"re": {"^привет$"},
			},
			input:   "other",
			message: "other does not match ^привет$",
		},
		{
			name: "unanchored regex matches substring",
			properties: map[string][]any{
				"re": {"привет"},
			},
			input: "prefix привет suffix",
		},
		{
			name: "empty regex",
			properties: map[string][]any{
				"re": {""},
			},
		},
		{
			name: "combined constraints",
			properties: map[string][]any{
				"min-length": {int64(6)},
				"max-length": {int64(6)},
				"re":         {"^привет$"},
			},
			input: "привет",
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewStr(test.properties)
			suite.Require().NoError(err)
			err = value.Validate(test.input)
			if test.message == "" {
				suite.NoError(err)
			} else {
				suite.EqualError(err, test.message)
			}
		})
	}
}

func (suite *StrTestSuite) TestPropertyArguments() {
	for _, property := range []struct {
		name     string
		value    any
		typeName string
	}{
		{
			name:     "min-length",
			value:    int64(1),
			typeName: "int64",
		},
		{
			name:     "max-length",
			value:    int64(1),
			typeName: "int64",
		},
		{
			name:     "re",
			value:    "привет",
			typeName: "string",
		},
	} {
		suite.Run(property.name, func() {
			for _, test := range []struct {
				name    string
				args    []any
				message string
				want    error
			}{
				{
					name: "missing",
					want: utils.ErrEmpty,
				},
				{
					name:    "multiple",
					args:    []any{property.value, property.value},
					message: "expected 1 element, got 2",
				},
				{
					name:    "wrong type",
					args:    []any{true},
					message: "expected " + property.typeName + " parameter, got bool",
				},
			} {
				suite.Run(test.name, func() {
					value, err := values.NewStr(map[string][]any{
						property.name: test.args,
					})
					suite.Require().Error(err)
					suite.Nil(value)
					suite.ErrorContains(err, "cannot add validator "+property.name+":")
					if test.want != nil {
						suite.ErrorIs(err, test.want)
					} else {
						suite.ErrorContains(err, test.message)
					}
				})
			}
		})
	}
}

func (suite *StrTestSuite) TestInvalidProperties() {
	for _, test := range []struct {
		name     string
		property string
		args     []any
		message  string
		want     error
	}{
		{
			name:     "negative minimum",
			property: "min-length",
			args:     []any{int64(-1)},
			message:  "length must be positive, not -1",
		},
		{
			name:     "negative maximum",
			property: "max-length",
			args:     []any{int64(-1)},
			message:  "length must be positive, not -1",
		},
		{
			name:     "invalid regex",
			property: "re",
			args:     []any{"["},
			message:  "incorrect regular expression [:",
		},
		{
			name:     "unknown property",
			property: "unknown",
			want:     values.ErrUnknownProperty,
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewStr(map[string][]any{
				test.property: test.args,
			})
			suite.Require().Error(err)
			suite.Nil(value)
			if test.want != nil {
				suite.ErrorIs(err, test.want)
			} else {
				suite.ErrorContains(err, test.message)
			}
			if test.property == "re" {
				var syntaxError *syntax.Error
				suite.ErrorAs(err, &syntaxError)
			}
		})
	}
}

func (suite *StrTestSuite) TestStringAndCompletion() {
	value, err := values.NewStr(map[string][]any{
		"re":         {"^привет$"},
		"min-length": {int64(1)},
		"max-length": {int64(6)},
	})
	suite.Require().NoError(err)
	suite.Equal("str(checks=max-length:6, min-length:1, re:^привет$)", value.String())
	for _, input := range []string{"", "привет", "other"} {
		suite.Run(input, func() {
			completions, directive := value.Complete(input)
			suite.Nil(completions)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func TestStr(t *testing.T) {
	suite.Run(t, &StrTestSuite{})
}
