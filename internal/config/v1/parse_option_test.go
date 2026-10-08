package v1_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParseOptionTestSuite struct {
	BaseTestSuite
}

func (suite *ParseOptionTestSuite) TestName() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "single name", doc: `option "name"`},
		{name: "unicode name", doc: `option "имя"`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().NoError(err)
			suite.NotNil(conf)
		})
	}
}

func (suite *ParseOptionTestSuite) TestInvalidName() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "no name", doc: `option`},
		{name: "multiple names", doc: `option "name" "other"`},
		{name: "integer", doc: `option 42`},
		{name: "float", doc: `option 1.5`},
		{name: "boolean", doc: `option #true`},
		{name: "null", doc: `option #null`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, "cannot process node option: cannot parse name")
		})
	}
}

func (suite *ParseOptionTestSuite) TestShort() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "single character", doc: `option "name" short="n"`},
		{name: "unicode character", doc: `option "name" short="я"`},
		{name: "distinct aliases", doc: "option \"name\" short=\"n\"\noption \"other\" short=\"o\""},
		{name: "multiple options without aliases", doc: "option \"name\"\noption \"other\""},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().NoError(err)
			suite.NotNil(conf)
		})
	}
}

func (suite *ParseOptionTestSuite) TestInvalidShort() {
	for _, test := range []struct {
		name string
		doc  string
		want string
	}{
		{name: "integer", doc: `option "name" short=42`, want: "value of 'short' must be string"},
		{name: "float", doc: `option "name" short=1.5`, want: "value of 'short' must be string"},
		{
			name: "boolean", doc: `option "name" short=#true`,
			want: "value of 'short' must be string",
		},
		{name: "null", doc: `option "name" short=#null`, want: "value of 'short' must be string"},
		{
			name: "empty string", doc: `option "name" short=""`,
			want: "length of 'short' must be 1, not 0",
		},
		{
			name: "multiple characters", doc: `option "name" short="nm"`,
			want: "length of 'short' must be 1, not 2",
		},
		{
			name: "multiple unicode characters", doc: `option "name" short="яя"`,
			want: "length of 'short' must be 1, not 2",
		},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, "cannot process node option: "+test.want)
		})
	}
}

func (suite *ParseOptionTestSuite) TestDuplicateShort() {
	conf, err := suite.Parse("option \"name\" short=\"n\"\noption \"other\" short=\"n\"")
	suite.Require().Error(err)
	suite.Nil(conf)
	suite.ErrorContains(err, "short option n is defined in both")
	suite.ErrorContains(err, "name")
	suite.ErrorContains(err, "other")
}

func (suite *ParseOptionTestSuite) TestChildFields() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "description", doc: `option "name" { description "An option"; }`},
		{name: "empty description", doc: `option "name" { description ""; }`},
		{name: "min-count", doc: `option "name" { min-count 0; }`},
		{name: "max-count", doc: `option "name" { max-count 4096; }`},
		{name: "string value", doc: `option "name" { value "str"; }`},
		{name: "integer value", doc: `option "name" { value "int"; }`},
		{name: "min-length", doc: `option "name" { value "str" { min-length 0; }; }`},
		{name: "max-length", doc: `option "name" { value "str" { max-length 4096; }; }`},
		{name: "regular expression", doc: `option "name" { value "str" { re ".*"; }; }`},
		{name: "minimum value", doc: `option "name" { value "int" { min -100; }; }`},
		{name: "maximum value", doc: `option "name" { value "int" { max 100; }; }`},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().NoError(err)
			suite.NotNil(conf)
		})
	}
}

func (suite *ParseOptionTestSuite) TestInvalidChildFields() {
	for _, test := range []struct {
		name string
		doc  string
		want string
	}{
		{
			name: "unknown child", doc: `option "name" { unknown; }`,
			want: "cannot parse unknown node: unknown node type",
		},
		{
			name: "description missing argument", doc: `option "name" { description; }`,
			want: "cannot parse description: expected 1 argument, got 0",
		},
		{
			name: "description extra argument", doc: `option "name" { description "a" "b"; }`,
			want: "cannot parse description: expected 1 argument, got 2",
		},
		{
			name: "description wrong type", doc: `option "name" { description 42; }`,
			want: "cannot parse description: unexpected value of type int64, expected string",
		},
		{
			name: "min-count missing argument", doc: `option "name" { min-count; }`,
			want: "cannot parse min-count: expected 1 argument, got 0",
		},
		{
			name: "min-count extra argument", doc: `option "name" { min-count 0 1; }`,
			want: "cannot parse min-count: expected 1 argument, got 2",
		},
		{
			name: "min-count wrong type", doc: `option "name" { min-count "0"; }`,
			want: "cannot parse min-count: unexpected value of type string, expected int64",
		},
		{
			name: "min-count float", doc: `option "name" { min-count 1.5; }`,
			want: "cannot parse min-count: unexpected value of type float64, expected int64",
		},
		{
			name: "max-count missing argument", doc: `option "name" { max-count; }`,
			want: "cannot parse max-count: expected 1 argument, got 0",
		},
		{
			name: "max-count extra argument", doc: `option "name" { max-count 1 2; }`,
			want: "cannot parse max-count: expected 1 argument, got 2",
		},
		{
			name: "max-count wrong type", doc: `option "name" { max-count "1"; }`,
			want: "cannot parse max-count: unexpected value of type string, expected int64",
		},
		{
			name: "max-count float", doc: `option "name" { max-count 1.5; }`,
			want: "cannot parse max-count: unexpected value of type float64, expected int64",
		},
		{
			name: "value missing type", doc: `option "name" { value; }`,
			want: "cannot parse value type: expected 1 argument, got 0",
		},
		{
			name: "value extra type", doc: `option "name" { value "str" "int"; }`,
			want: "cannot parse value type: expected 1 argument, got 2",
		},
		{
			name: "value wrong type", doc: `option "name" { value 42; }`,
			want: "cannot parse value type: unexpected value of type int64, expected string",
		},
		{
			name: "value parameter missing argument", doc: `option "name" { value "str" { min-length; }; }`,
			want: "cannot parse argument of value min-length: expected 1 argument, got 0",
		},
		{
			name: "value parameter extra argument", doc: `option "name" { value "str" { min-length 0 1; }; }`,
			want: "cannot parse argument of value min-length: expected 1 argument, got 2",
		},
	} {
		suite.Run(test.name, func() {
			conf, err := suite.Parse(test.doc)
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, "cannot process node option: cannot parse option name:")
			suite.ErrorContains(err, test.want)
		})
	}
}

func TestParseOption(t *testing.T) {
	suite.Run(t, &ParseOptionTestSuite{})
}
