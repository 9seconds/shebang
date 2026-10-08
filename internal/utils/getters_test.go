package utils_test

import (
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/stretchr/testify/suite"
)

type GettersTestSuite struct {
	suite.Suite
}

func (suite *GettersTestSuite) TestAllStrings() {
	for _, test := range []struct {
		name    string
		values  []any
		want    []string
		message string
	}{
		{
			name: "nil input",
			want: []string{},
		},
		{
			name:   "empty input",
			values: []any{},
			want:   []string{},
		},
		{
			name:   "single string",
			values: []any{"привет"},
			want:   []string{"привет"},
		},
		{
			name:   "order and empty strings preserved",
			values: []any{"second", "", "привет", "first"},
			want:   []string{"second", "", "привет", "first"},
		},
		{
			name:    "wrong type first",
			values:  []any{int64(1), "text"},
			message: "expected string parameter, got int64",
		},
		{
			name:    "wrong type after valid value",
			values:  []any{"text", true},
			message: "expected string parameter, got bool",
		},
		{
			name:    "nil value",
			values:  []any{nil},
			message: "expected string parameter, got <nil>",
		},
	} {
		suite.Run(test.name, func() {
			got, err := utils.All[string](test.values)
			if test.message == "" {
				suite.Require().NoError(err)
				suite.Equal(test.want, got)
			} else {
				suite.EqualError(err, test.message)
				suite.Nil(got)
			}
		})
	}
}

func (suite *GettersTestSuite) TestAllIntegers() {
	for _, test := range []struct {
		name    string
		values  []any
		want    []int64
		message string
	}{
		{
			name:   "int64 values",
			values: []any{int64(-1), int64(0), int64(2)},
			want:   []int64{-1, 0, 2},
		},
		{
			name:    "int is not converted to int64",
			values:  []any{1},
			message: "expected int64 parameter, got int",
		},
		{
			name:    "float is not converted to int64",
			values:  []any{float64(1)},
			message: "expected int64 parameter, got float64",
		},
	} {
		suite.Run(test.name, func() {
			got, err := utils.All[int64](test.values)
			if test.message == "" {
				suite.Require().NoError(err)
				suite.Equal(test.want, got)
			} else {
				suite.EqualError(err, test.message)
				suite.Nil(got)
			}
		})
	}
}

func (suite *GettersTestSuite) TestOneStrings() {
	for _, test := range []struct {
		name    string
		values  []any
		want    string
		message string
		empty   bool
	}{
		{
			name:  "nil input",
			empty: true,
		},
		{
			name:   "empty input",
			values: []any{},
			empty:  true,
		},
		{
			name:   "single string",
			values: []any{"привет"},
			want:   "привет",
		},
		{
			name:   "empty string is a value",
			values: []any{""},
		},
		{
			name:    "two values",
			values:  []any{"first", "second"},
			message: "expected 1 element, got 2",
		},
		{
			name:    "three values",
			values:  []any{"first", "second", "third"},
			message: "expected 1 element, got 3",
		},
		{
			name:    "wrong type",
			values:  []any{int64(1)},
			message: "expected string parameter, got int64",
		},
		{
			name:    "type error precedes count error",
			values:  []any{"first", true},
			message: "expected string parameter, got bool",
		},
		{
			name:    "nil value",
			values:  []any{nil},
			message: "expected string parameter, got <nil>",
		},
	} {
		suite.Run(test.name, func() {
			got, err := utils.One[string](test.values)
			if test.empty {
				suite.ErrorIs(err, utils.ErrEmpty)
				suite.Empty(got)
			} else if test.message != "" {
				suite.EqualError(err, test.message)
				suite.Empty(got)
			} else {
				suite.Require().NoError(err)
				suite.Equal(test.want, got)
			}
		})
	}
}

func (suite *GettersTestSuite) TestOneIntegers() {
	for _, test := range []struct {
		name    string
		values  []any
		want    int64
		message string
	}{
		{
			name:   "zero is a value",
			values: []any{int64(0)},
		},
		{
			name:   "negative value",
			values: []any{int64(-2)},
			want:   -2,
		},
		{
			name:    "int is not converted to int64",
			values:  []any{2},
			message: "expected int64 parameter, got int",
		},
	} {
		suite.Run(test.name, func() {
			got, err := utils.One[int64](test.values)
			if test.message == "" {
				suite.Require().NoError(err)
				suite.Equal(test.want, got)
			} else {
				suite.EqualError(err, test.message)
				suite.Zero(got)
			}
		})
	}
}

func (suite *GettersTestSuite) TestPointers() {
	value := new("привет")
	for _, test := range []struct {
		name    string
		value   any
		want    *string
		message string
	}{
		{
			name:  "pointer identity preserved",
			value: value,
			want:  value,
		},
		{
			name:  "typed nil is a value",
			value: (*string)(nil),
		},
		{
			name:    "untyped nil fails",
			message: "expected *string parameter, got <nil>",
		},
	} {
		suite.Run(test.name, func() {
			all, allErr := utils.All[*string]([]any{test.value})
			one, oneErr := utils.One[*string]([]any{test.value})
			if test.message != "" {
				suite.EqualError(allErr, test.message)
				suite.EqualError(oneErr, test.message)
				suite.Nil(all)
				suite.Nil(one)
				return
			}
			suite.Require().NoError(allErr)
			suite.Require().NoError(oneErr)
			suite.Require().Len(all, 1)
			if test.want == nil {
				suite.Nil(all[0])
				suite.Nil(one)
			} else {
				suite.Same(test.want, all[0])
				suite.Same(test.want, one)
			}
		})
	}
}

func TestGetters(t *testing.T) {
	suite.Run(t, &GettersTestSuite{})
}
