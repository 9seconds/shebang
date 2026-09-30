package v1_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParseExecuteTestSuite struct {
	BaseTestSuite
}

func (suite *ParseExecuteTestSuite) TestStringArguments() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "single argument", doc: `execute "bash"`},
		{name: "multiple arguments", doc: `execute "bash" "-e" "-u"`},
		{name: "argument containing spaces", doc: `execute "bash -e -u"`},
		{name: "empty string argument", doc: `execute "bash" ""`},
		{name: "escaped characters", doc: `execute "bash" "echo \"hello\"\\world"`},
		{name: "unicode argument", doc: `execute "bash" "привет"`},
		{name: "no arguments", doc: `execute`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().NoError(err)
			suite.NotNil(conf)
		})
	}
}

func (suite *ParseExecuteTestSuite) TestInvalidArgumentTypes() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "integer", doc: `execute 42`},
		{name: "float", doc: `execute 1.5`},
		{name: "boolean", doc: `execute #true`},
		{name: "null", doc: `execute #null`},
		{name: "invalid first argument", doc: `execute 42 "bash"`},
		{name: "invalid middle argument", doc: `execute "bash" 42 "-e"`},
		{name: "invalid last argument", doc: `execute "bash" 42`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, "cannot process node execute: unexpected value of type")
			suite.ErrorContains(err, "expected string")
		})
	}
}

func TestParseExecute(t *testing.T) {
	suite.Run(t, &ParseExecuteTestSuite{})
}
