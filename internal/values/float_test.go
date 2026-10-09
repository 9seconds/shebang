package values_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type FloatTestSuite struct {
	suite.Suite
}

func (suite *FloatTestSuite) TestParsing() {
	for _, test := range []struct {
		name  string
		input string
		want  error
	}{
		{
			name:  "integer spelling",
			input: "12",
		},
		{
			name:  "signed fraction",
			input: "-1.25",
		},
		{
			name:  "leading plus and zeros",
			input: "+001.25",
		},
		{
			name:  "fraction without integer part",
			input: ".5",
		},
		{
			name:  "scientific notation",
			input: "1.25e2",
		},
		{
			name:  "hexadecimal float",
			input: "0x1.8p+1",
		},
		{
			name:  "digit separators",
			input: "1_000.5",
		},
		{
			name:  "largest finite value",
			input: strconv.FormatFloat(math.MaxFloat64, 'g', -1, 64),
		},
		{
			name:  "smallest positive value",
			input: strconv.FormatFloat(math.SmallestNonzeroFloat64, 'g', -1, 64),
		},
		{
			name:  "negative zero",
			input: "-0.0",
		},
		{
			name:  "underflow rounds to zero",
			input: "1e-4000",
		},
		{
			name:  "overflow",
			input: "1e4000",
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
			name:  "spaces",
			input: " 1.5 ",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "NaN",
			input: "NaN",
			want:  values.ErrNonFiniteFloat,
		},
		{
			name:  "positive infinity",
			input: "+Inf",
			want:  values.ErrNonFiniteFloat,
		},
		{
			name:  "negative infinity",
			input: "-Infinity",
			want:  values.ErrNonFiniteFloat,
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewFloat(nil)
			suite.Require().NoError(err)
			suite.Equal("float(checks=)", value.String())

			err = value.Validate(test.input)
			if test.want == nil {
				suite.Require().NoError(err)

				return
			}

			suite.Require().ErrorIs(err, test.want)
		})
	}
}

func (suite *FloatTestSuite) TestBounds() {
	for _, test := range []struct {
		name     string
		property string
		bound    float64
		input    string
		want     *values.NumConstraintError[float64]
		message  string
	}{
		{
			name:     "minimum inclusive",
			property: "min",
			bound:    1.25,
			input:    "1.25",
		},
		{
			name:     "below negative minimum",
			property: "min",
			bound:    -1.25,
			input:    "-1.5",
			want:     values.NewNumConstraintError(values.NumConstraintMin, -1.25, -1.5),
			message:  "number must be at least -1.25, got -1.5",
		},
		{
			name:     "maximum inclusive",
			property: "max",
			bound:    1.25,
			input:    "1.25",
		},
		{
			name:     "above maximum",
			property: "max",
			bound:    1.25,
			input:    "1.5",
			want:     values.NewNumConstraintError(values.NumConstraintMax, 1.25, 1.5),
			message:  "number must be at most 1.25, got 1.5",
		},
		{
			name:     "zero bound accepts negative zero",
			property: "min",
			input:    "-0.0",
		},
		{
			name:     "positive minimum rejects underflow",
			property: "min",
			bound:    math.SmallestNonzeroFloat64,
			input:    "1e-4000",
			want: values.NewNumConstraintError(
				values.NumConstraintMin, math.SmallestNonzeroFloat64, 0.0),
			message: "number must be at least 5e-324, got 0",
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewFloat(map[string][]any{
				test.property: {test.bound},
			})
			suite.Require().NoError(err)

			err = value.Validate(test.input)
			if test.want == nil {
				suite.Require().NoError(err)

				return
			}

			var constraintError *values.NumConstraintError[float64]

			suite.Require().ErrorAs(err, &constraintError)
			suite.Require().EqualError(err, test.message)
			suite.Equal(*test.want, *constraintError)
			suite.Equal(test.message, test.want.Error())
		})
	}
}

func (suite *FloatTestSuite) TestProperties() {
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
					args: []any{1.0, 2.0},
					want: utils.ErrSingleArgumentExpected,
				},
				{
					name:      "integer is not a float bound",
					args:      []any{int64(1)},
					typeError: true,
				},
				{
					name:      "string is not converted",
					args:      []any{"1.0"},
					typeError: true,
				},
				{
					name: "NaN bound",
					args: []any{math.NaN()},
					want: values.ErrNonFiniteFloat,
				},
				{
					name: "positive infinite bound",
					args: []any{math.Inf(1)},
					want: values.ErrNonFiniteFloat,
				},
				{
					name: "negative infinite bound",
					args: []any{math.Inf(-1)},
					want: values.ErrNonFiniteFloat,
				},
			} {
				suite.Run(test.name, func() {
					value, err := values.NewFloat(map[string][]any{
						property: test.args,
					})
					suite.Require().ErrorContains(err, "cannot add validator "+property+":")
					suite.Nil(value)

					if test.typeError {
						var typeError *utils.ArgumentTypeError

						suite.Require().ErrorAs(err, &typeError)
						suite.Equal("float64", typeError.Expected)
					} else {
						suite.Require().ErrorIs(err, test.want)
					}
				})
			}
		})
	}

	value, err := values.NewFloat(map[string][]any{
		"unknown": {1.0},
	})
	suite.Require().ErrorIs(err, values.ErrUnknownProperty)
	suite.Nil(value)
}

func (suite *FloatTestSuite) TestCombinedBoundsAndCompletion() {
	value, err := values.New("float", map[string][]any{
		"min": {0.5},
		"max": {0.5},
	})
	suite.Require().NoError(err)
	suite.Equal("float(checks=max:0.5, min:0.5)", value.String())
	suite.Require().NoError(value.Validate("5e-1"))
	suite.Require().Error(value.Validate("0.49"))
	suite.Require().Error(value.Validate("0.51"))
	suite.Require().ErrorIs(value.Validate("NaN"), values.ErrNonFiniteFloat)

	for _, prefix := range []string{"", "привет", "1.5", "NaN", "-"} {
		completions, directive := value.Complete(prefix)
		suite.Nil(completions)
		suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
	}
}

func TestFloat(t *testing.T) {
	t.Parallel()

	suite.Run(t, &FloatTestSuite{})
}
