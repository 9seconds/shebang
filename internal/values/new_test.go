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
			want: "str(subvalidators=)",
		},
		{
			name: "default ignores properties",
			properties: map[string][]any{
				"unknown": {true},
			},
			want: "str(subvalidators=)",
		},
		{
			name:      "string type",
			valueType: "str",
			want:      "str(subvalidators=)",
		},
		{
			name:      "string properties forwarded",
			valueType: "str",
			properties: map[string][]any{
				"min-length": {int64(2)},
			},
			want: "str(subvalidators=min-length:2)",
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
			err: values.ErrUnknownProperty,
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.New(test.valueType, nil, test.properties)
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
	} {
		suite.Run(test.name, func() {
			value, err := values.New(test.name, nil, nil)
			suite.Require().NoError(err)

			candidates, directive := value.Complete(test.input)
			suite.Equal([]cobra.Completion{test.input}, candidates)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
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
	} {
		suite.Run(test.name, func() {
			for _, properties := range []map[string]any{nil, {}} {
				value, err := test.constructor(properties, nil)
				suite.Require().NoError(err)
				suite.NotNil(value)
			}

			for _, property := range []any{true, false, nil} {
				properties := map[string]any{"xx": property}
				value, err := test.constructor(properties, nil)
				suite.Require().ErrorIs(err, values.ErrNoProperties)
				suite.Nil(value)

				value, err = values.New(test.name, properties, nil)
				suite.Require().ErrorIs(err, values.ErrNoProperties)
				suite.Nil(value)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	suite.Run(t, &NewTestSuite{})
}
