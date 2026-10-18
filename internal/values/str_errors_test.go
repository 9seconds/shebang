package values_test

import (
	"testing"

	"github.com/9seconds/shebang/internal/values"
	"github.com/stretchr/testify/suite"
)

type StrErrorsTestSuite struct {
	suite.Suite
}

func (suite *StrErrorsTestSuite) TestNegativeLengthError() {
	for _, property := range []string{
		"min-length",
		"max-length",
	} {
		suite.Run(property, func() {
			// Arrange
			var lengthError *values.ErrStrNegativeLength

			// Act
			constructed := values.NewErrStrNegativeLength(-2)
			value, err := values.NewStr(nil, map[string][]any{
				property: {int64(-2)},
			})

			// Assert
			suite.Nil(value)
			suite.Equal(int64(-2), constructed.Length)
			suite.Equal("length must be non-negative, not -2", constructed.Error())
			suite.Require().ErrorAs(err, &lengthError)
			suite.Equal(*constructed, *lengthError)
			suite.Require().ErrorContains(err, "cannot add validator "+property+":")
		})
	}
}

func (suite *StrErrorsTestSuite) TestLengthConstraintError() {
	for _, test := range []struct {
		name     string
		property string
		kind     values.StrLengthConstraintKind
		bound    int
		message  string
	}{
		{
			name:     "minimum counts unicode runes",
			property: "min-length",
			kind:     values.StrLengthConstraintMinimum,
			bound:    7,
			message:  "minimum length must be at least 7, got 6",
		},
		{
			name:     "maximum counts unicode runes",
			property: "max-length",
			kind:     values.StrLengthConstraintMaximum,
			bound:    5,
			message:  "maximum length must be at most 5, got 6",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewStr(nil, map[string][]any{
				test.property: {int64(test.bound)},
			})
			suite.Require().NoError(err)

			var lengthError *values.ErrStrLengthConstraint

			// Act
			constructed := values.NewErrStrLengthConstraint(test.kind, test.bound, 6)
			err = value.Validate("привет")

			// Assert
			suite.Equal(values.ErrStrLengthConstraint{
				Kind:     test.kind,
				Expected: test.bound,
				Actual:   6,
			}, *constructed)
			suite.Equal(test.message, constructed.Error())
			suite.Require().ErrorAs(err, &lengthError)
			suite.Require().EqualError(err, test.message)
			suite.Equal(*constructed, *lengthError)
		})
	}
}

func (suite *StrErrorsTestSuite) TestRegexMismatchError() {
	// Arrange
	value, err := values.NewStr(nil, map[string][]any{
		"re": {"^other$"},
	})
	suite.Require().NoError(err)

	var mismatchError *values.ErrStrRegexMismatch

	// Act
	constructed := values.NewErrStrRegexMismatch("привет", "^other$")
	err = value.Validate("привет")

	// Assert
	suite.Equal("привет", constructed.Value)
	suite.Equal("^other$", constructed.Pattern)
	suite.Equal("привет does not match ^other$", constructed.Error())
	suite.Require().ErrorAs(err, &mismatchError)
	suite.Equal(*constructed, *mismatchError)
}

func TestStrErrors(t *testing.T) {
	t.Parallel()

	suite.Run(t, &StrErrorsTestSuite{})
}
