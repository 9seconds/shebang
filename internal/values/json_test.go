package values_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type JSONTestSuite struct {
	suite.Suite
}

func (suite *JSONTestSuite) TestValidValues() {
	for _, test := range []struct {
		name  string
		input string
	}{
		{
			name:  "null",
			input: "null",
		},
		{
			name:  "true",
			input: "true",
		},
		{
			name:  "false",
			input: "false",
		},
		{
			name:  "integer",
			input: "1",
		},
		{
			name:  "negative zero",
			input: "-0",
		},
		{
			name:  "fraction and exponent",
			input: "-1.25e+12",
		},
		{
			name:  "number outside float64 range",
			input: "1e1000",
		},
		{
			name:  "large integer",
			input: "1234567890123456789012345678901234567890",
		},
		{
			name:  "empty string",
			input: `""`,
		},
		{
			name:  "Unicode string",
			input: `"привет"`,
		},
		{
			name:  "escaped string",
			input: `"\u043f\n\"\\"`,
		},
		{
			name:  "empty array",
			input: "[]",
		},
		{
			name:  "empty object",
			input: "{}",
		},
		{
			name:  "nested mixed values",
			input: `{"items":[null,true,1,"привет",{"nested":[]}]}`,
		},
		{
			name:  "surrounding whitespace",
			input: " \t\r\n{ \"x\": 1 }\n ",
		},
		{
			name:  "duplicate object names",
			input: `{"x":1,"x":2}`,
		},
		{
			name:  "unpaired surrogate follows encoding/json semantics",
			input: `"\ud800"`,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.New("json", nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			suite.Require().NoError(err)
			suite.Equal("json()", value.String())
			suite.Equal([]cobra.Completion{test.input}, candidates)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func (suite *JSONTestSuite) TestSyntaxErrors() {
	for _, test := range []struct {
		name    string
		input   string
		offset  int64
		message string
	}{
		{
			name:    "empty",
			message: "unexpected end of JSON input",
		},
		{
			name:    "whitespace only",
			input:   " \t\n",
			offset:  3,
			message: "unexpected end of JSON input",
		},
		{
			name:    "incomplete object",
			input:   "{",
			offset:  1,
			message: "unexpected end of JSON input",
		},
		{
			name:    "trailing value",
			input:   "1 2",
			offset:  3,
			message: "after top-level value",
		},
		{
			name:    "missing object value",
			input:   `{"a":}`,
			offset:  6,
			message: "looking for beginning of value",
		},
		{
			name:  "trailing array comma",
			input: "[1,]",
		},
		{
			name:  "unquoted object name",
			input: "{a:1}",
		},
		{
			name:  "invalid escape",
			input: `"\x"`,
		},
		{
			name:  "unterminated string",
			input: `"hello`,
		},
		{
			name:  "unescaped control character",
			input: "\"hello\nworld\"",
		},
		{
			name:  "leading zero",
			input: "01",
		},
		{
			name:  "leading plus",
			input: "+1",
		},
		{
			name:  "missing exponent digits",
			input: "1e",
		},
		{
			name:  "NaN",
			input: "NaN",
		},
		{
			name:  "comment",
			input: "/* comment */ null",
		},
		{
			name:  "mismatched delimiter",
			input: "[1}",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewJSON(nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			var syntaxError *json.SyntaxError

			suite.Require().ErrorAs(err, &syntaxError)
			suite.NotEmpty(syntaxError.Error())
			suite.Empty(candidates)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if test.message != "" {
				suite.Equal(test.offset, syntaxError.Offset)
				suite.Contains(syntaxError.Error(), test.message)
			}
		})
	}
}

func (suite *JSONTestSuite) TestNestingBoundary() {
	for _, depth := range []int{9999, 10000, 10001} {
		for _, object := range []bool{false, true} {
			suite.Run(fmt.Sprintf("depth=%d/object=%t", depth, object), func() {
				// Arrange
				opening, closing := "[", "]"
				if object {
					opening, closing = `{"x":`, "}"
				}

				input := strings.Repeat(opening, depth) + "null" + strings.Repeat(closing, depth)
				value, err := values.NewJSON(nil, nil)
				suite.Require().NoError(err)

				// Act
				err = value.Validate(input)

				// Assert
				if depth <= 10000 {
					suite.Require().NoError(err)
				} else {
					var syntaxError *json.SyntaxError

					suite.Require().ErrorAs(err, &syntaxError)
					suite.Contains(syntaxError.Error(), "exceeded max depth")
				}
			})
		}
	}
}

func (suite *JSONTestSuite) TestConfiguration() {
	// Arrange
	properties := map[string]any{}
	subvalidators := map[string][]any{}

	// Act
	value, err := values.NewJSON(properties, subvalidators)

	// Assert
	suite.Require().NoError(err)
	suite.Require().NotNil(value)
	suite.Equal("json()", value.String())

	for _, args := range []any{nil, int64(1)} {
		suite.Run(fmt.Sprintf("unsupported subvalidator/%v", args), func() {
			// Arrange
			subvalidators := map[string][]any{
				"schema": {args},
			}

			// Act
			value, err := values.New("json", nil, subvalidators)

			// Assert
			suite.Nil(value)
			suite.Require().ErrorIs(err, values.ErrNoSubvalidators)
		})
	}
}

func TestJSON(t *testing.T) {
	t.Parallel()

	suite.Run(t, &JSONTestSuite{})
}
