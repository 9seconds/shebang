package validators_test

import (
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/validators"
	"github.com/stretchr/testify/suite"
)

type StringValidatorTestSuite struct {
	suite.Suite
}

func (suite *StringValidatorTestSuite) TestDefaults() {
	for _, properties := range []map[string]any{nil, {}} {
		validator, err := validators.NewStringValidator(properties)
		suite.Require().NoError(err)
		suite.Require().NotNil(validator)
		for _, value := range []string{"", "hello", "привет", "a\nb", strings.Repeat("a", 4097)} {
			suite.NoError(validator.Validate(value))
		}
	}
}

func (suite *StringValidatorTestSuite) TestLengthBoundaries() {
	for _, test := range []struct {
		name  string
		check string
		limit int64
		value string
		want  string
	}{
		{name: "minimum zero accepts empty", check: "min-length", limit: 0},
		{name: "below minimum", check: "min-length", limit: 3, value: "ab", want: "min-length must have at least 3 characters"},
		{name: "at minimum", check: "min-length", limit: 3, value: "abc"},
		{name: "above minimum", check: "min-length", limit: 3, value: "abcd"},
		{name: "unicode below minimum", check: "min-length", limit: 3, value: "яя", want: "min-length must have at least 3 characters"},
		{name: "unicode at minimum", check: "min-length", limit: 3, value: "я😀界"},
		{name: "maximum zero accepts empty", check: "max-length", limit: 0},
		{name: "maximum zero rejects nonempty", check: "max-length", limit: 0, value: "a", want: "max-length must have at most 0 characters"},
		{name: "below maximum", check: "max-length", limit: 3, value: "ab"},
		{name: "at maximum", check: "max-length", limit: 3, value: "abc"},
		{name: "above maximum", check: "max-length", limit: 3, value: "abcd", want: "max-length must have at most 3 characters"},
		{name: "unicode at maximum", check: "max-length", limit: 3, value: "я😀界"},
		{name: "unicode above maximum", check: "max-length", limit: 3, value: "я😀界a", want: "max-length must have at most 3 characters"},
	} {
		suite.Run(test.name, func() {
			validator, err := validators.NewStringValidator(map[string]any{test.check: test.limit})
			suite.Require().NoError(err)
			err = validator.Validate(test.value)
			if test.want == "" {
				suite.NoError(err)
			} else {
				suite.EqualError(err, test.want)
			}
		})
	}
}

func (suite *StringValidatorTestSuite) TestRegexp() {
	for _, test := range []struct {
		name  string
		check string
		expr  string
		value string
		want  string
	}{
		{name: "scheme check name", check: "re", expr: "^[a-z]+$", value: "hello"},
		{name: "nonmatching", check: "re", expr: "^[a-z]+$", value: "123", want: "re does not match ^[a-z]+$"},
		{name: "anchors reject surrounding text", check: "re", expr: "^hello$", value: "say hello", want: "re does not match ^hello$"},
		{name: "unanchored expression", check: "re", expr: "hello", value: "say hello world"},
		{name: "empty expression", check: "re"},
		{name: "default expression accepts empty", check: "re", expr: ".*"},
		{name: "default expression accepts multiline", check: "re", expr: ".*", value: "a\nb"},
		{name: "unicode expression", check: "re", expr: "^привет$", value: "привет"},
	} {
		suite.Run(test.name, func() {
			validator, err := validators.NewStringValidator(map[string]any{test.check: test.expr})
			suite.Require().NoError(err)
			err = validator.Validate(test.value)
			if test.want == "" {
				suite.NoError(err)
			} else {
				suite.EqualError(err, test.want)
			}
		})
	}
}

func (suite *StringValidatorTestSuite) TestInvalidChecks() {
	for _, test := range []struct {
		name  string
		check string
		arg   any
		want  string
	}{
		{name: "negative minimum", check: "min-length", arg: int64(-1), want: "min-length must be a non-negative integer"},
		{name: "minimum string", check: "min-length", arg: "1", want: "min-length must be a non-negative integer"},
		{name: "minimum float", check: "min-length", arg: 1.5, want: "min-length must be a non-negative integer"},
		{name: "minimum boolean", check: "min-length", arg: true, want: "min-length must be a non-negative integer"},
		{name: "minimum null", check: "min-length", want: "min-length must be a non-negative integer"},
		{name: "negative maximum", check: "max-length", arg: int64(-1), want: "max-length must be a non-negative integer"},
		{name: "maximum string", check: "max-length", arg: "1", want: "max-length must be a non-negative integer"},
		{name: "maximum float", check: "max-length", arg: 1.5, want: "max-length must be a non-negative integer"},
		{name: "maximum boolean", check: "max-length", arg: true, want: "max-length must be a non-negative integer"},
		{name: "maximum null", check: "max-length", want: "max-length must be a non-negative integer"},
		{name: "regexp integer", check: "re", arg: int64(1), want: "re must be a string"},
		{name: "regexp boolean", check: "re", arg: true, want: "re must be a string"},
		{name: "regexp null", check: "re", want: "re must be a string"},
		{name: "invalid regexp", check: "re", arg: "[", want: "re is invalid regexp:"},
		{name: "unsupported regexp check", check: "regexp", arg: ".*", want: "unknown validator regexp"},
		{name: "unknown check", check: "unknown", arg: int64(1), want: "unknown validator unknown"},
	} {
		suite.Run(test.name, func() {
			validator, err := validators.NewStringValidator(map[string]any{test.check: test.arg})
			suite.Require().Error(err)
			suite.Nil(validator)
			suite.ErrorContains(err, test.want)

			validator, err = validators.NewStringValidator(nil)
			suite.Require().NoError(err)
			err = validator.AddCheck(test.check, test.arg)
			suite.Require().Error(err)
			suite.ErrorContains(err, test.want)
			suite.NoError(validator.Validate("anything"))
		})
	}
}

func (suite *StringValidatorTestSuite) TestCombinedChecks() {
	validator, err := validators.NewStringValidator(map[string]any{
		"min-length": int64(2),
		"max-length": int64(4),
		"re":         "^[a-z]+$",
	})
	suite.Require().NoError(err)
	suite.NoError(validator.Validate("ab"))
	suite.NoError(validator.Validate("abcd"))
	suite.EqualError(validator.Validate("a"), "min-length must have at least 2 characters")
	suite.EqualError(validator.Validate("abcde"), "max-length must have at most 4 characters")
	err = validator.Validate("1")
	suite.Require().Error(err)
}

func (suite *StringValidatorTestSuite) TestAddCheck() {
	validator, err := validators.NewStringValidator(nil)
	suite.Require().NoError(err)
	suite.NoError(validator.Validate("a"))
	suite.Require().NoError(validator.AddCheck("min-length", int64(2)))
	suite.EqualError(validator.Validate("a"), "min-length must have at least 2 characters")
	suite.NoError(validator.Validate("ab"))
	suite.Require().NoError(validator.AddCheck("max-length", int64(3)))
	suite.EqualError(validator.Validate("abcd"), "max-length must have at most 3 characters")
	suite.Require().NoError(validator.AddCheck("re", "^[a-z]+$"))
	suite.EqualError(validator.Validate("12"), "re does not match ^[a-z]+$")
	suite.NoError(validator.Validate("abc"))
}

func TestStringValidator(t *testing.T) {
	suite.Run(t, &StringValidatorTestSuite{})
}
