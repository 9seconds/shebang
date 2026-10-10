package values_test

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type IntTestSuite struct {
	suite.Suite
}

func (suite *IntTestSuite) TestParsing() {
	for _, test := range []struct {
		name  string
		input string
		want  error
	}{
		{
			name:  "zero",
			input: "0",
		},
		{
			name:  "positive sign",
			input: "+12",
		},
		{
			name:  "negative integer",
			input: "-12",
		},
		{
			name:  "leading zeros are decimal",
			input: "0012",
		},
		{
			name:  "minimum int64",
			input: strconv.FormatInt(math.MinInt64, 10),
		},
		{
			name:  "maximum int64",
			input: strconv.FormatInt(math.MaxInt64, 10),
		},
		{
			name:  "positive overflow",
			input: strconv.FormatUint(uint64(math.MaxInt64)+1, 10),
			want:  strconv.ErrRange,
		},
		{
			name:  "negative overflow",
			input: "-" + strconv.FormatUint(uint64(math.MaxInt64)+2, 10),
			want:  strconv.ErrRange,
		},
		{
			name: "empty",
			want: strconv.ErrSyntax,
		},
		{
			name:  "unicode text",
			input: "привет",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "decimal fraction",
			input: "1.5",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "hexadecimal prefix",
			input: "0x10",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "spaces",
			input: " 12 ",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "digit separator",
			input: "1_000",
			want:  strconv.ErrSyntax,
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewInt(nil)
			suite.Require().NoError(err)

			err = value.Validate(test.input)
			if test.want == nil {
				suite.Require().NoError(err)

				return
			}

			suite.Require().ErrorIs(err, test.want)

			var parseError *strconv.NumError

			suite.Require().ErrorAs(err, &parseError)
			suite.Equal(test.input, parseError.Num)
		})
	}
}

func (suite *IntTestSuite) TestBounds() {
	for _, test := range []struct {
		name     string
		property string
		bound    int64
		input    string
		want     *values.NumConstraintError[int64]
		message  string
	}{
		{
			name:     "minimum inclusive",
			property: "min",
			bound:    2,
			input:    "2",
		},
		{
			name:     "above minimum",
			property: "min",
			bound:    2,
			input:    "3",
		},
		{
			name:     "below negative minimum",
			property: "min",
			bound:    -2,
			input:    "-3",
			want:     values.NewNumConstraintError(values.NumConstraintMin, int64(-2), int64(-3)),
			message:  "number must be at least -2, got -3",
		},
		{
			name:     "maximum inclusive",
			property: "max",
			bound:    2,
			input:    "2",
		},
		{
			name:     "below maximum",
			property: "max",
			bound:    2,
			input:    "1",
		},
		{
			name:     "above negative maximum",
			property: "max",
			bound:    -2,
			input:    "-1",
			want:     values.NewNumConstraintError(values.NumConstraintMax, int64(-2), int64(-1)),
			message:  "number must be at most -2, got -1",
		},
		{
			name:     "zero minimum",
			property: "min",
			input:    "0",
		},
		{
			name:     "minimum int64 bound",
			property: "min",
			bound:    math.MinInt64,
			input:    strconv.FormatInt(math.MinInt64, 10),
		},
		{
			name:     "maximum int64 bound",
			property: "max",
			bound:    math.MaxInt64,
			input:    strconv.FormatInt(math.MaxInt64, 10),
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewInt(map[string][]any{
				test.property: {test.bound},
			})
			suite.Require().NoError(err)

			err = value.Validate(test.input)
			if test.want == nil {
				suite.Require().NoError(err)

				return
			}

			suite.Require().EqualError(err, test.message)

			var constraintError *values.NumConstraintError[int64]

			suite.Require().ErrorAs(err, &constraintError)
			suite.Equal(*test.want, *constraintError)
			suite.Equal(test.message, test.want.Error())
		})
	}
}

func (suite *IntTestSuite) TestProperties() {
	for _, property := range []string{"min", "max"} {
		suite.Run(property, func() {
			for _, test := range []struct {
				name      string
				args      []any
				want      error
				typeError bool
			}{
				{
					name: "missing",
					want: utils.ErrEmpty,
				},
				{
					name: "multiple",
					args: []any{int64(1), int64(2)},
					want: utils.ErrSingleArgumentExpected,
				},
				{
					name:      "string is not converted",
					args:      []any{"1"},
					typeError: true,
				},
				{
					name:      "float is not converted",
					args:      []any{float64(1)},
					typeError: true,
				},
			} {
				suite.Run(test.name, func() {
					value, err := values.NewInt(map[string][]any{
						property: test.args,
					})
					suite.Require().ErrorContains(err, "cannot add validator "+property+":")
					suite.Nil(value)

					if test.typeError {
						var typeError *utils.ArgumentTypeError

						suite.Require().ErrorAs(err, &typeError)
						suite.Equal("int64", typeError.Expected)
					} else {
						suite.Require().ErrorIs(err, test.want)
					}
				})
			}
		})
	}

	value, err := values.NewInt(map[string][]any{
		"unknown": {int64(1)},
	})
	suite.Require().ErrorIs(err, values.ErrUnknownProperty)
	suite.Nil(value)
}

func (suite *IntTestSuite) TestCombinedBoundsAndCompletion() {
	for _, bound := range []int64{-2, 0, 2} {
		suite.Run(strconv.FormatInt(bound, 10), func() {
			value, err := values.New("int", map[string][]any{
				"min": {bound},
				"max": {bound},
			})
			suite.Require().NoError(err)
			suite.Equal(fmt.Sprintf("int(subvalidators=max:%d, min:%d)", bound, bound), value.String())
			suite.Require().NoError(value.Validate(strconv.FormatInt(bound, 10)))
			suite.Require().Error(value.Validate(strconv.FormatInt(bound-1, 10)))
			suite.Require().Error(value.Validate(strconv.FormatInt(bound+1, 10)))

			for _, prefix := range []string{"", "привет", "1", "-"} {
				completions, directive := value.Complete(prefix)
				suite.Nil(completions)
				suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
			}
		})
	}
}

func TestInt(t *testing.T) {
	t.Parallel()

	suite.Run(t, &IntTestSuite{})
}
