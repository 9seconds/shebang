package values_test

import (
	"testing"

	"github.com/9seconds/shebang/internal/values"
	"github.com/stretchr/testify/suite"
)

type ValueErrorsTestSuite struct {
	suite.Suite
}

func (suite *ValueErrorsTestSuite) TestNegativeLengthError() {
	for _, property := range []string{"min-length", "max-length"} {
		suite.Run(property, func() {
			constructed := values.NewNegativeLengthError(-2)
			suite.Equal(int64(-2), constructed.Length)
			suite.Equal("length must be positive, not -2", constructed.Error())

			value, err := values.NewStr(map[string][]any{
				property: {int64(-2)},
			})
			suite.Nil(value)

			var lengthError *values.NegativeLengthError

			suite.Require().ErrorAs(err, &lengthError)
			suite.Equal(*constructed, *lengthError)
			suite.Require().ErrorContains(err, "cannot add validator "+property+":")
		})
	}
}

func (suite *ValueErrorsTestSuite) TestLengthConstraintError() {
	for _, test := range []struct {
		name     string
		property string
		kind     values.LengthConstraintKind
		bound    int
		message  string
	}{
		{
			name:     "minimum counts unicode runes",
			property: "min-length",
			kind:     values.LengthConstraintMinimum,
			bound:    7,
			message:  "minimum length must be at least 7, got 6",
		},
		{
			name:     "maximum counts unicode runes",
			property: "max-length",
			kind:     values.LengthConstraintMaximum,
			bound:    5,
			message:  "minimum length must be at most 5, got 6",
		},
	} {
		suite.Run(test.name, func() {
			constructed := values.NewLengthConstraintError(test.kind, test.bound, 6)
			suite.Equal(values.LengthConstraintError{
				Kind:     test.kind,
				Expected: test.bound,
				Actual:   6,
			}, *constructed)
			suite.Equal(test.message, constructed.Error())

			value, err := values.NewStr(map[string][]any{
				test.property: {int64(test.bound)},
			})
			suite.Require().NoError(err)

			var lengthError *values.LengthConstraintError

			err = value.Validate("привет")
			suite.Require().ErrorAs(err, &lengthError)
			suite.Require().EqualError(err, test.message)
			suite.Equal(*constructed, *lengthError)
		})
	}
}

func (suite *ValueErrorsTestSuite) TestRegexMismatchError() {
	constructed := values.NewRegexMismatchError("привет", "^other$")
	suite.Equal("привет", constructed.Value)
	suite.Equal("^other$", constructed.Pattern)
	suite.Equal("привет does not match ^other$", constructed.Error())

	value, err := values.NewStr(map[string][]any{
		"re": {"^other$"},
	})
	suite.Require().NoError(err)

	var mismatchError *values.RegexMismatchError

	err = value.Validate("привет")
	suite.Require().ErrorAs(err, &mismatchError)
	suite.Equal(*constructed, *mismatchError)
}

func TestValueErrors(t *testing.T) {
	t.Parallel()

	suite.Run(t, &ValueErrorsTestSuite{})
}
