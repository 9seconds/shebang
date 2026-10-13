package values_test

import (
	"encoding/base64"
	"fmt"
	"math"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type Base64TestSuite struct {
	suite.Suite
}

func (suite *Base64TestSuite) TestFormats() {
	for _, format := range []struct {
		name     string
		encoding *base64.Encoding
	}{
		{
			name:     "std",
			encoding: base64.StdEncoding,
		},
		{
			name:     "url",
			encoding: base64.URLEncoding,
		},
		{
			name:     "raw",
			encoding: base64.RawStdEncoding,
		},
		{
			name:     "raw-url",
			encoding: base64.RawURLEncoding,
		},
	} {
		suite.Run(format.name, func() {
			for _, payload := range []struct {
				name string
				data []byte
			}{
				{
					name: "empty",
				},
				{
					name: "one byte",
					data: []byte("f"),
				},
				{
					name: "two bytes",
					data: []byte("fo"),
				},
				{
					name: "full quantum",
					data: []byte("foo"),
				},
				{
					name: "unicode",
					data: []byte("привет"),
				},
				{
					name: "binary and alphabet symbols",
					data: []byte{0xfb, 0xff, 0x00, 0xff},
				},
			} {
				suite.Run(payload.name, func() {
					// Arrange
					input := format.encoding.EncodeToString(payload.data)
					length := int64(len(payload.data))
					value, err := values.NewBase64(map[string]any{
						"format": format.name,
					}, map[string][]any{
						"min-bytes": {length},
						"max-bytes": {length},
					})
					suite.Require().NoError(err)

					// Act
					err = value.Validate(input)
					candidates, directive := value.Complete(input)

					// Assert
					suite.Require().NoError(err)
					suite.Equal([]cobra.Completion{input}, candidates)
					suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
				})
			}
		})
	}
}

func (suite *Base64TestSuite) TestDecode() {
	for _, test := range []struct {
		name   string
		format string
		input  string
		valid  bool
	}{
		{
			name:  "default standard padding",
			input: "Zg==",
			valid: true,
		},
		{
			name:  "default standard alphabet",
			input: "+/8=",
			valid: true,
		},
		{
			name:   "URL alphabet",
			format: "url",
			input:  "-_8=",
			valid:  true,
		},
		{
			name:   "raw standard alphabet",
			format: "raw",
			input:  "+/8",
			valid:  true,
		},
		{
			name:   "raw URL alphabet",
			format: "raw-url",
			input:  "-_8",
			valid:  true,
		},
		{
			name:  "CRLF ignored",
			input: "Z\r\ng==\n",
			valid: true,
		},
		{
			name:  "noncanonical trailing bits accepted",
			input: "Zh==",
			valid: true,
		},
		{
			name:  "missing padding",
			input: "Zg",
		},
		{
			name:   "raw rejects padding",
			format: "raw",
			input:  "Zg==",
		},
		{
			name:  "standard rejects URL alphabet",
			input: "-_8=",
		},
		{
			name:   "URL rejects standard alphabet",
			format: "url",
			input:  "+/8=",
		},
		{
			name:   "raw standard rejects URL alphabet",
			format: "raw",
			input:  "-_8",
		},
		{
			name:   "raw URL rejects standard alphabet",
			format: "raw-url",
			input:  "+/8",
		},
		{
			name:   "raw URL rejects padding",
			format: "raw-url",
			input:  "-_8=",
		},
		{
			name:   "URL requires padding",
			format: "url",
			input:  "-_8",
		},
		{
			name:  "leading space",
			input: " Zg==",
		},
		{
			name:  "tab",
			input: "Z\tg==",
		},
		{
			name:  "invalid symbol",
			input: "Z!==",
		},
		{
			name:  "invalid padding position",
			input: "Z=g=",
		},
		{
			name:  "excess padding",
			input: "Zg===",
		},
		{
			name:  "data after padding",
			input: "Zg==Zm8=",
		},
		{
			name:   "incomplete raw quantum",
			format: "raw",
			input:  "Z",
		},
		{
			name:  "literal Unicode",
			input: "привет",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			properties := map[string]any{}
			if test.format != "" {
				properties["format"] = test.format
			}

			value, err := values.NewBase64(properties, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			suite.Equal("base64()", value.String())
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if test.valid {
				suite.Require().NoError(err)
				suite.Equal([]cobra.Completion{test.input}, candidates)
			} else {
				var corrupt base64.CorruptInputError

				suite.Require().ErrorAs(err, &corrupt)
				suite.Empty(candidates)
			}
		})
	}
}

func (suite *Base64TestSuite) TestConfigurationErrors() {
	for _, test := range []struct {
		name       string
		properties map[string]any
		want       error
		typeError  bool
		format     string
	}{
		{
			name: "unknown property",
			properties: map[string]any{
				"unknown": true,
			},
			want: values.ErrUnknownProperty,
		},
		{
			name: "nonstring format",
			properties: map[string]any{
				"format": int64(1),
			},
			typeError: true,
		},
		{
			name: "null format",
			properties: map[string]any{
				"format": nil,
			},
			typeError: true,
		},
		{
			name: "empty format",
			properties: map[string]any{
				"format": "",
			},
		},
		{
			name: "case sensitive format",
			properties: map[string]any{
				"format": "STD",
			},
			format: "STD",
		},
		{
			name: "unknown format",
			properties: map[string]any{
				"format": "привет",
			},
			format: "привет",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			properties := test.properties

			// Act
			value, err := values.NewBase64(properties, nil)

			// Assert
			suite.Nil(value)
			suite.Require().ErrorContains(err, "incorrect property")

			switch {
			case test.want != nil:
				suite.Require().ErrorIs(err, test.want)
			case test.typeError:
				var typeError *utils.ArgumentTypeError

				suite.Require().ErrorAs(err, &typeError)
				suite.Equal("string", typeError.Expected)
			default:
				var formatError *values.ErrBase64Format

				suite.Require().ErrorAs(err, &formatError)
				suite.Equal(test.format, formatError.Format)
				suite.Equal("unknown format "+test.format, formatError.Error())
			}
		})
	}
}

func (suite *Base64TestSuite) TestSubvalidatorErrors() {
	for _, name := range []string{
		"min-bytes",
		"max-bytes",
	} {
		suite.Run(name, func() {
			for _, test := range []struct {
				name      string
				args      []any
				want      error
				typeError bool
			}{
				{
					name: "missing bound",
					want: utils.ErrEmpty,
				},
				{
					name: "multiple bounds",
					args: []any{int64(1), int64(2)},
					want: utils.ErrSingleArgumentExpected,
				},
				{
					name: "negative bound",
					args: []any{int64(-1)},
					want: values.ErrBase64NegativeLength,
				},
				{
					name:      "string bound",
					args:      []any{"1"},
					typeError: true,
				},
				{
					name:      "float bound",
					args:      []any{1.0},
					typeError: true,
				},
				{
					name:      "boolean bound",
					args:      []any{true},
					typeError: true,
				},
				{
					name:      "null bound",
					args:      []any{nil},
					typeError: true,
				},
			} {
				suite.Run(test.name, func() {
					// Arrange
					subvalidators := map[string][]any{
						name: test.args,
					}

					// Act
					value, err := values.NewBase64(nil, subvalidators)

					// Assert
					suite.Nil(value)
					suite.Require().ErrorContains(err, "incorrect subvalidator "+name)

					if test.typeError {
						var typeError *utils.ArgumentTypeError

						suite.Require().ErrorAs(err, &typeError)
						suite.Equal("int64", typeError.Expected)
					} else {
						suite.Require().ErrorIs(err, test.want)
					}
				})
			}
		})
	}

	// Arrange
	subvalidators := map[string][]any{
		"unknown": {int64(1)},
	}

	// Act
	value, err := values.NewBase64(nil, subvalidators)

	// Assert
	suite.Nil(value)
	suite.Require().ErrorIs(err, values.ErrUnknownSubvalidator)
}

func (suite *Base64TestSuite) TestByteBounds() {
	for _, test := range []struct {
		name     string
		bound    int64
		length   int
		relation string
	}{
		{
			name:     "min-bytes",
			bound:    2,
			length:   1,
			relation: "at least",
		},
		{
			name:   "min-bytes",
			bound:  2,
			length: 2,
		},
		{
			name:   "min-bytes",
			bound:  2,
			length: 3,
		},
		{
			name:   "max-bytes",
			bound:  2,
			length: 1,
		},
		{
			name:   "max-bytes",
			bound:  2,
			length: 2,
		},
		{
			name:     "max-bytes",
			bound:    2,
			length:   3,
			relation: "at most",
		},
		{
			name: "min-bytes",
		},
		{
			name: "max-bytes",
		},
		{
			name:     "max-bytes",
			length:   1,
			relation: "at most",
		},
		{
			name:     "min-bytes",
			bound:    math.MaxInt64,
			length:   1,
			relation: "at least",
		},
		{
			name:   "max-bytes",
			bound:  math.MaxInt64,
			length: 1,
		},
	} {
		suite.Run(fmt.Sprintf("%s/%d/%d", test.name, test.bound, test.length), func() {
			// Arrange
			input := base64.StdEncoding.EncodeToString(make([]byte, test.length))
			value, err := values.NewBase64(nil, map[string][]any{
				test.name: {test.bound},
			})
			suite.Require().NoError(err)

			// Act
			err = value.Validate(input)
			candidates, directive := value.Complete(input)

			// Assert
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if test.relation == "" {
				suite.Require().NoError(err)
				suite.Equal([]cobra.Completion{input}, candidates)
			} else {
				var lengthError *values.ErrBase64LengthConstraint

				suite.Require().ErrorAs(err, &lengthError)
				//nolint:testifylint // Expected is the actual error field being verified.
				suite.Equal(test.bound, lengthError.Expected)
				suite.Equal(int64(test.length), lengthError.Actual)
				suite.Equal(test.relation, lengthError.Relation)
				suite.Equal(fmt.Sprintf("length must be %s %d bytes", test.relation, test.bound),
					lengthError.Error())
				suite.Empty(candidates)
			}
		})
	}
}

func (suite *Base64TestSuite) TestContradictoryBounds() {
	for _, input := range []string{
		"",
		"Zg==",
		"Zm8=",
		"Zm9v",
	} {
		suite.Run(input, func() {
			// Arrange
			value, err := values.NewBase64(nil, map[string][]any{
				"min-bytes": {int64(2)},
				"max-bytes": {int64(1)},
			})
			suite.Require().NoError(err)

			// Act
			err = value.Validate(input)

			// Assert
			var lengthError *values.ErrBase64LengthConstraint

			suite.Require().ErrorAs(err, &lengthError)
			suite.Equal("base64(max-bytes:1, min-bytes:2)", value.String())
		})
	}
}

func TestBase64(t *testing.T) {
	t.Parallel()

	suite.Run(t, &Base64TestSuite{})
}
