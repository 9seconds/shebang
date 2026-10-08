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
	for _, properties := range []map[string][]any{nil, {}} {
		validator, err := validators.New("str", properties)
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
		{
			name: "below minimum", check: "min-length", limit: 3, value: "ab",
			want: "min-length must have at least 3 characters",
		},
		{name: "at minimum", check: "min-length", limit: 3, value: "abc"},
		{name: "above minimum", check: "min-length", limit: 3, value: "abcd"},
		{
			name: "unicode below minimum", check: "min-length", limit: 3, value: "яя",
			want: "min-length must have at least 3 characters",
		},
		{name: "unicode at minimum", check: "min-length", limit: 3, value: "я😀界"},
		{name: "maximum zero accepts empty", check: "max-length", limit: 0},
		{
			name: "maximum zero rejects nonempty", check: "max-length", limit: 0, value: "a",
			want: "max-length must have at most 0 characters",
		},
		{name: "below maximum", check: "max-length", limit: 3, value: "ab"},
		{name: "at maximum", check: "max-length", limit: 3, value: "abc"},
		{
			name: "above maximum", check: "max-length", limit: 3, value: "abcd",
			want: "max-length must have at most 3 characters",
		},
		{name: "unicode at maximum", check: "max-length", limit: 3, value: "я😀界"},
		{
			name: "unicode above maximum", check: "max-length", limit: 3, value: "я😀界a",
			want: "max-length must have at most 3 characters",
		},
	} {
		suite.Run(test.name, func() {
			validator, err := validators.New("str", map[string][]any{
				test.check: {test.limit},
			})
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
			validator, err := validators.New("str", map[string][]any{
				test.check: {test.expr},
			})
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
		{
			name: "negative minimum", check: "min-length", arg: int64(-1),
			want: "min-length must be a non-negative integer",
		},
		{
			name: "minimum string", check: "min-length", arg: "1",
			want: "expected int64 parameter, but got string",
		},
		{
			name: "minimum float", check: "min-length", arg: 1.5,
			want: "expected int64 parameter, but got float64",
		},
		{
			name: "minimum boolean", check: "min-length", arg: true,
			want: "expected int64 parameter, but got bool",
		},
		{name: "minimum null", check: "min-length", want: "expected int64 parameter, but got <nil>"},
		{
			name: "negative maximum", check: "max-length", arg: int64(-1),
			want: "max-length must be a non-negative integer",
		},
		{
			name: "maximum string", check: "max-length", arg: "1",
			want: "expected int64 parameter, but got string",
		},
		{
			name: "maximum float", check: "max-length", arg: 1.5,
			want: "expected int64 parameter, but got float64",
		},
		{
			name: "maximum boolean", check: "max-length", arg: true,
			want: "expected int64 parameter, but got bool",
		},
		{name: "maximum null", check: "max-length", want: "expected int64 parameter, but got <nil>"},
		{
			name: "regexp integer", check: "re", arg: int64(1),
			want: "expected string parameter, but got int64",
		},
		{name: "regexp boolean", check: "re", arg: true, want: "expected string parameter, but got bool"},
		{name: "regexp null", check: "re", want: "expected string parameter, but got <nil>"},
		{name: "invalid regexp", check: "re", arg: "[", want: "re is invalid regexp:"},
		{name: "unsupported regexp check", check: "regexp", arg: ".*", want: "unknown validator regexp"},
		{name: "unknown check", check: "unknown", arg: int64(1), want: "unknown validator unknown"},
	} {
		suite.Run(test.name, func() {
			validator, err := validators.New("str", map[string][]any{
				test.check: {test.arg},
			})
			suite.Require().Error(err)
			suite.Nil(validator)
			suite.ErrorContains(err, test.want)
		})
	}
}

func (suite *StringValidatorTestSuite) TestCombinedChecks() {
	validator, err := validators.New("str", map[string][]any{
		"min-length": {int64(2)},
		"max-length": {int64(4)},
		"re":         {"^[a-z]+$"},
	})
	suite.Require().NoError(err)
	suite.NoError(validator.Validate("ab"))
	suite.NoError(validator.Validate("abcd"))
	suite.EqualError(validator.Validate("a"), "min-length must have at least 2 characters")
	suite.EqualError(validator.Validate("abcde"), "max-length must have at most 4 characters")
	err = validator.Validate("1")
	suite.Require().Error(err)
}

func (suite *StringValidatorTestSuite) TestArgumentCount() {
	for _, check := range []string{"min-length", "max-length", "re"} {
		suite.Run(check, func() {
			value, valueType := any(int64(1)), "int64"
			if check == "re" {
				value, valueType = ".*", "string"
			}
			for _, args := range [][]any{nil, {}, {value, value}} {
				validator, err := validators.New("str", map[string][]any{check: args})
				suite.Require().Error(err)
				suite.Nil(validator)
				if len(args) == 0 {
					suite.EqualError(err, "1 value of "+valueType+" must be defined")
				} else {
					suite.EqualError(err, "expected 1 parameter of "+valueType+" but got 2")
				}
			}
		})
	}
}

func (suite *StringValidatorTestSuite) TestString() {
	validator, err := validators.New("str", nil)
	suite.Require().NoError(err)
	suite.Equal("str(checks=)", validator.String())

	validator, err = validators.New("str", map[string][]any{
		"min-length": {int64(2)},
		"max-length": {int64(4)},
		"re":         {"^[a-z]+$"},
	})
	suite.Require().NoError(err)
	suite.Contains(validator.String(), "max-length:4")
	suite.Contains(validator.String(), "min-length:2")
	suite.Contains(validator.String(), "re:^[a-z]+$")
}

func (suite *StringValidatorTestSuite) TestUnknownType() {
	validator, err := validators.New("unknown", nil)
	suite.Nil(validator)
	suite.EqualError(err, "unknown validator type unknown")
}

func TestStringValidator(t *testing.T) {
	suite.Run(t, &StringValidatorTestSuite{})
}
