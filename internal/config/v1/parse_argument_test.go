package v1_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParseArgumentTestSuite struct {
	BaseTestSuite
}

func (suite *ParseArgumentTestSuite) TestChildFields() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "no children", doc: `argument`},
		{name: "scheme node", doc: `argument "str" min=0 max=0 {}`},
		{name: "description", doc: `argument { description "An argument"; }`},
		{name: "empty description", doc: `argument { description ""; }`},
		{name: "min-count", doc: `argument { min-count 0; }`},
		{name: "max-count", doc: `argument { max-count 4096; }`},
		{name: "string value", doc: `argument { value "str"; }`},
		{name: "integer value", doc: `argument { value "int"; }`},
		{name: "min-length", doc: `argument { value "str" { min-length 0; }; }`},
		{name: "max-length", doc: `argument { value "str" { max-length 4096; }; }`},
		{name: "regular expression", doc: `argument { value "str" { re ".*"; }; }`},
		{name: "minimum value", doc: `argument { value "int" { min -100; }; }`},
		{name: "maximum value", doc: `argument { value "int" { max 100; }; }`},
		{name: "value parameter without arguments", doc: `argument { value "str" { min-length; }; }`},
		{name: "value parameter with multiple arguments", doc: `argument { value "str" { min-length 0 1; }; }`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().NoError(err)
			suite.NotNil(conf)
		})
	}
}

func (suite *ParseArgumentTestSuite) TestInvalidChildFields() {
	for _, test := range []struct {
		name string
		doc  string
		want string
	}{
		{
			name: "unknown child", doc: `argument { unknown; }`,
			want: "cannot parse unknown node: unknown node type",
		},
		{
			name: "description missing argument", doc: `argument { description; }`,
			want: "cannot parse description: expected 1 argument, got 0",
		},
		{
			name: "description extra argument", doc: `argument { description "a" "b"; }`,
			want: "cannot parse description: expected 1 argument, got 2",
		},
		{
			name: "description wrong type", doc: `argument { description 42; }`,
			want: "cannot parse description: unexpected value of type int64, expected string",
		},
		{
			name: "min-count missing argument", doc: `argument { min-count; }`,
			want: "cannot parse min-count: expected 1 argument, got 0",
		},
		{
			name: "min-count extra argument", doc: `argument { min-count 0 1; }`,
			want: "cannot parse min-count: expected 1 argument, got 2",
		},
		{
			name: "min-count wrong type", doc: `argument { min-count "0"; }`,
			want: "cannot parse min-count: unexpected value of type string, expected int64",
		},
		{
			name: "min-count float", doc: `argument { min-count 1.5; }`,
			want: "cannot parse min-count: unexpected value of type float64, expected int64",
		},
		{
			name: "max-count missing argument", doc: `argument { max-count; }`,
			want: "cannot parse max-count: expected 1 argument, got 0",
		},
		{
			name: "max-count extra argument", doc: `argument { max-count 1 2; }`,
			want: "cannot parse max-count: expected 1 argument, got 2",
		},
		{
			name: "max-count wrong type", doc: `argument { max-count "1"; }`,
			want: "cannot parse max-count: unexpected value of type string, expected int64",
		},
		{
			name: "max-count float", doc: `argument { max-count 1.5; }`,
			want: "cannot parse max-count: unexpected value of type float64, expected int64",
		},
		{
			name: "value missing type", doc: `argument { value; }`,
			want: "cannot parse value type: expected 1 argument, got 0",
		},
		{
			name: "value extra type", doc: `argument { value "str" "int"; }`,
			want: "cannot parse value type: expected 1 argument, got 2",
		},
		{
			name: "value wrong type", doc: `argument { value 42; }`,
			want: "cannot parse value type: unexpected value of type int64, expected string",
		},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, "cannot process node argument: cannot parse argument:")
			suite.ErrorContains(err, test.want)
		})
	}
}

func TestParseArgument(t *testing.T) {
	suite.Run(t, &ParseArgumentTestSuite{})
}
