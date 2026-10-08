package v1_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParseExampleTestSuite struct {
	BaseTestSuite
}

func (suite *ParseExampleTestSuite) TestStringArgument() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "single argument", doc: `example "script"`},
		{name: "argument containing spaces", doc: `example "script --name value"`},
		{name: "empty string argument", doc: `example ""`},
		{name: "escaped characters", doc: `example "echo \"hello\"\\world"`},
		{name: "unicode argument", doc: `example "привет"`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().NoError(err)
			suite.NotNil(conf)
		})
	}
}

func (suite *ParseExampleTestSuite) TestInvalidArgumentTypes() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "integer", doc: `example 42`},
		{name: "float", doc: `example 1.5`},
		{name: "boolean", doc: `example #true`},
		{name: "null", doc: `example #null`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, "cannot process node example: unexpected value of type")
			suite.ErrorContains(err, "expected string")
		})
	}
}

func (suite *ParseExampleTestSuite) TestInvalidArgumentCount() {
	for _, test := range []struct {
		name string
		doc  string
		want string
	}{
		{name: "no arguments", doc: `example`, want: "expected 1 argument, got 0"},
		{name: "multiple arguments", doc: `example "script" "--help"`, want: "expected 1 argument, got 2"},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, "cannot process node example: "+test.want)
		})
	}
}

func TestParseExample(t *testing.T) {
	suite.Run(t, &ParseExampleTestSuite{})
}
