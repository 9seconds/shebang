package v1_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParseDescriptionTestSuite struct {
	BaseTestSuite
}

func (suite *ParseDescriptionTestSuite) TestStringArgument() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "single argument", doc: `description "script"`},
		{name: "argument containing spaces", doc: `description "A script description"`},
		{name: "empty string argument", doc: `description ""`},
		{name: "escaped characters", doc: `description "echo \"hello\"\\world"`},
		{name: "unicode argument", doc: `description "привет"`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().NoError(err)
			suite.NotNil(conf)
		})
	}
}

func (suite *ParseDescriptionTestSuite) TestInvalidArgumentTypes() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "integer", doc: `description 42`},
		{name: "float", doc: `description 1.5`},
		{name: "boolean", doc: `description #true`},
		{name: "null", doc: `description #null`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, "cannot process node description: unexpected value of type")
			suite.ErrorContains(err, "expected string")
		})
	}
}

func (suite *ParseDescriptionTestSuite) TestInvalidArgumentCount() {
	for _, test := range []struct {
		name string
		doc  string
		want string
	}{
		{name: "no arguments", doc: `description`, want: "expected 1 argument, got 0"},
		{name: "multiple arguments", doc: `description "A script" "description"`, want: "expected 1 argument, got 2"},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, "cannot process node description: "+test.want)
		})
	}
}

func TestParseDescription(t *testing.T) {
	suite.Run(t, &ParseDescriptionTestSuite{})
}
