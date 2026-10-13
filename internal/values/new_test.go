package values_test

import (
	"testing"

	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type NewTestSuite struct {
	suite.Suite
}

func (suite *NewTestSuite) TestNew() {
	for _, test := range []struct {
		name       string
		valueType  string
		properties map[string][]any
		want       string
		err        error
	}{
		{
			name: "default type",
			want: "str()",
		},
		{
			name: "default ignores properties",
			properties: map[string][]any{
				"unknown": {true},
			},
			want: "str()",
		},
		{
			name:      "string type",
			valueType: "str",
			want:      "str()",
		},
		{
			name:      "string properties forwarded",
			valueType: "str",
			properties: map[string][]any{
				"min-length": {int64(2)},
			},
			want: "str(min-length:2)",
		},
		{
			name:      "unknown type",
			valueType: "unknown",
			err:       values.ErrUnknownValueType,
		},
		{
			name:      "type is case sensitive",
			valueType: "STR",
			err:       values.ErrUnknownValueType,
		},
		{
			name:      "invalid string property",
			valueType: "str",
			properties: map[string][]any{
				"unknown": {},
			},
			err: values.ErrUnknownSubvalidator,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			properties := test.properties

			// Act
			value, err := values.New(test.valueType, nil, properties)

			// Assert
			if test.err != nil {
				suite.Require().ErrorIs(err, test.err)
				suite.Nil(value)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(value)
				suite.Equal(test.want, value.String())
				suite.NoError(value.Validate("привет"))
			}
		})
	}
}

func (suite *NewTestSuite) TestValidInputCompletion() {
	for _, test := range []struct {
		name  string
		input string
	}{
		{
			name:  "str",
			input: "привет",
		},
		{
			name:  "int",
			input: "+0012",
		},
		{
			name:  "float",
			input: "5e-1",
		},
		{
			name:  "ip",
			input: "2001:DB8::1",
		},
		{
			name:  "port",
			input: "08080",
		},
		{
			name:  "base64",
			input: "0L/RgNC40LLQtdGC",
		},
		{
			name:  "uuid",
			input: "F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6",
		},
		{
			name:  "datetime",
			input: "2026-10-12T13:14:15.123+02:00",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.New(test.name, nil, nil)
			suite.Require().NoError(err)

			// Act
			candidates, directive := value.Complete(test.input)

			// Assert
			suite.Equal([]cobra.Completion{test.input}, candidates)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func (suite *NewTestSuite) TestDatetimeConfiguration() {
	// Arrange
	value, err := values.New("datetime", map[string]any{
		"format": "unix",
	}, map[string][]any{
		"date-after":  {"100"},
		"date-before": {"200"},
	})
	suite.Require().NoError(err)

	for _, test := range []struct {
		input string
		valid bool
	}{
		{
			input: "+00150",
			valid: true,
		},
		{
			input: "100",
		},
		{
			input: "200",
		},
		{
			input: "2026-10-12",
		},
	} {
		suite.Run(test.input, func() {
			// Act
			err := value.Validate(test.input)

			// Assert
			suite.Equal("datetime(date-after:100, date-before:200)", value.String())

			if test.valid {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *NewTestSuite) TestBase64Configuration() {
	// Arrange
	properties := map[string]any{
		"format": "raw-url",
	}
	subvalidators := map[string][]any{
		"min-bytes": {int64(2)},
		"max-bytes": {int64(2)},
	}

	// Act
	value, err := values.New("base64", properties, subvalidators)

	// Assert
	suite.Require().NoError(err)
	suite.Equal("base64(max-bytes:2, min-bytes:2)", value.String())
	suite.Require().NoError(value.Validate("-_8"))
	suite.Require().Error(value.Validate("-_8="))
	suite.Require().Error(value.Validate("Zg"))
	suite.Require().Error(value.Validate("Zm9v"))
}

func (suite *NewTestSuite) TestDatetimeConfigurationErrors() {
	for _, test := range []struct {
		name          string
		properties    map[string]any
		subvalidators map[string][]any
		want          error
	}{
		{
			name: "property",
			properties: map[string]any{
				"unknown": true,
			},
			want: values.ErrUnknownProperty,
		},
		{
			name: "subvalidator",
			subvalidators: map[string][]any{
				"unknown": {"1h"},
			},
			want: values.ErrUnknownSubvalidator,
		},
	} {
		suite.Run(test.name, func() {
			// Act
			value, err := values.New("datetime", test.properties, test.subvalidators)

			// Assert
			suite.Nil(value)
			suite.Require().ErrorIs(err, test.want)
		})
	}
}

func (suite *NewTestSuite) TestNodeProperties() {
	for _, test := range []struct {
		name        string
		constructor func(map[string]any, map[string][]any) (values.Value, error)
	}{
		{
			name:        "str",
			constructor: values.NewStr,
		},
		{
			name:        "int",
			constructor: values.NewInt,
		},
		{
			name:        "float",
			constructor: values.NewFloat,
		},
		{
			name:        "ip",
			constructor: values.NewIP,
		},
		{
			name:        "port",
			constructor: values.NewPort,
		},
		{
			name:        "enum",
			constructor: values.NewEnum,
		},
		{
			name:        "filepath",
			constructor: values.NewFilepath,
		},
		{
			name:        "uuid",
			constructor: values.NewUUID,
		},
	} {
		suite.Run(test.name, func() {
			for _, properties := range []map[string]any{nil, {}} {
				// Act
				value, err := test.constructor(properties, nil)

				// Assert
				suite.Require().NoError(err)
				suite.NotNil(value)
			}

			for _, property := range []any{true, false, nil} {
				// Arrange
				properties := map[string]any{
					"xx": property,
				}

				// Act
				value, err := test.constructor(properties, nil)
				factoryValue, factoryError := values.New(test.name, properties, nil)

				// Assert
				suite.Require().ErrorIs(err, values.ErrNoProperties)
				suite.Nil(value)

				suite.Require().ErrorIs(factoryError, values.ErrNoProperties)
				suite.Nil(factoryValue)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	suite.Run(t, &NewTestSuite{})
}
