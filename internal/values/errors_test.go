package values_test

import (
	"testing"

	"github.com/9seconds/shebang/internal/values"
	"github.com/stretchr/testify/suite"
)

type ErrorsTestSuite struct {
	suite.Suite
}

func (suite *ErrorsTestSuite) TestNumericBoundsAcrossTypes() {
	for _, valueType := range []string{
		"int",
		"float",
	} {
		suite.Run(valueType, func() {
			for _, test := range []struct {
				name  string
				input string
				valid bool
			}{
				{
					name:  "below lower boundary",
					input: "-2",
				},
				{
					name:  "lower boundary",
					input: "-1",
					valid: true,
				},
				{
					name:  "inside range",
					input: "0",
					valid: true,
				},
				{
					name:  "upper boundary",
					input: "1",
					valid: true,
				},
				{
					name:  "above upper boundary",
					input: "2",
				},
			} {
				suite.Run(test.name, func() {
					// Arrange
					bounds := map[string][]any{
						"min": {int64(-1)},
						"max": {int64(1)},
					}
					if valueType == "float" {
						bounds = map[string][]any{
							"min": {-1.0},
							"max": {1.0},
						}
					}

					value, err := values.New(valueType, nil, bounds)
					suite.Require().NoError(err)

					// Act
					err = value.Validate(test.input)

					// Assert
					if test.valid {
						suite.Require().NoError(err)
					} else {
						suite.Require().Error(err)
					}
				})
			}
		})
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()

	suite.Run(t, &ErrorsTestSuite{})
}
