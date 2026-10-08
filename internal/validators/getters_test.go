package validators

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type GetOneTestSuite struct {
	suite.Suite
}

func (suite *GetOneTestSuite) TestInt64() {
	for _, test := range []struct {
		name   string
		values []any
		want   int64
		err    string
	}{
		{name: "nil slice", err: "1 value of int64 must be defined"},
		{name: "empty slice", values: []any{}, err: "1 value of int64 must be defined"},
		{name: "positive", values: []any{int64(42)}, want: 42},
		{name: "zero", values: []any{int64(0)}},
		{name: "negative", values: []any{int64(-1)}, want: -1},
		{name: "int is not int64", values: []any{42}, err: "expected int64 parameter, but got int"},
		{name: "string", values: []any{"42"}, err: "expected int64 parameter, but got string"},
		{name: "nil value", values: []any{nil}, err: "expected int64 parameter, but got <nil>"},
		{
			name: "multiple values", values: []any{int64(1), int64(2)},
			err: "expected 1 parameter of int64 but got 2",
		},
		{
			name: "count checked before type", values: []any{"a", "b", "c"},
			err: "expected 1 parameter of int64 but got 3",
		},
	} {
		suite.Run(test.name, func() {
			value, err := getOne[int64](test.values)
			if test.err == "" {
				suite.Require().NoError(err)
			} else {
				suite.EqualError(err, test.err)
			}
			suite.Equal(test.want, value)
		})
	}
}

func (suite *GetOneTestSuite) TestString() {
	for _, test := range []struct {
		name   string
		values []any
		want   string
		err    string
	}{
		{name: "nil slice", err: "1 value of string must be defined"},
		{name: "empty slice", values: []any{}, err: "1 value of string must be defined"},
		{name: "empty string", values: []any{""}},
		{name: "value preserved", values: []any{" привет \n"}, want: " привет \n"},
		{name: "integer", values: []any{int64(42)}, err: "expected string parameter, but got int64"},
		{name: "nil value", values: []any{nil}, err: "expected string parameter, but got <nil>"},
		{
			name: "multiple values", values: []any{"a", "b"},
			err: "expected 1 parameter of string but got 2",
		},
	} {
		suite.Run(test.name, func() {
			value, err := getOne[string](test.values)
			if test.err == "" {
				suite.Require().NoError(err)
			} else {
				suite.EqualError(err, test.err)
			}
			suite.Equal(test.want, value)
		})
	}
}

func (suite *GetOneTestSuite) TestTypedNil() {
	var input *int
	value, err := getOne[*int]([]any{input})
	suite.NoError(err)
	suite.Nil(value)
}

func TestGetOne(t *testing.T) {
	suite.Run(t, &GetOneTestSuite{})
}
