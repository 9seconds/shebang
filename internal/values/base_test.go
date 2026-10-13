package values

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type BaseTestSuite struct {
	suite.Suite
}

func (suite *BaseTestSuite) TestValidate() {
	want := errors.New("invalid value")
	for _, test := range []struct {
		name                string
		prepareError        error
		subvalidatorError   error
		want                error
		subvalidatorsCalled int
	}{
		{
			name:         "preparation error skips subvalidators",
			prepareError: want,
			want:         want,
		},
		{
			name:                "subvalidator error propagated",
			subvalidatorError:   want,
			want:                want,
			subvalidatorsCalled: 1,
		},
		{
			name:                "prepared value passes subvalidators",
			subvalidatorsCalled: 1,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			prepareCalls := 0
			subvalidatorCalls := 0

			var raw, prepared string

			value := baseValue[string]{
				toValue: func(input string) (string, error) {
					prepareCalls++
					raw = input

					return strings.ToUpper(input), test.prepareError
				},
				subvalidators: map[string]func(string) error{
					"subvalidator": func(input string) error {
						subvalidatorCalls++
						prepared = input

						return test.subvalidatorError
					},
				},
			}

			// Act
			err := value.Validate("привет")

			// Assert
			if test.want != nil {
				suite.Require().ErrorIs(err, test.want)
			} else {
				suite.Require().NoError(err)
			}

			suite.Equal(1, prepareCalls)
			suite.Equal(test.subvalidatorsCalled, subvalidatorCalls)
			suite.Equal("привет", raw)

			if subvalidatorCalls > 0 {
				suite.Equal("ПРИВЕТ", prepared)
			}
		})
	}
}

func (suite *BaseTestSuite) TestAllSubvalidatorsRun() {
	// Arrange
	called := make(map[string]bool)
	value := baseValue[string]{
		toValue: func(input string) (string, error) {
			return input, nil
		},
		subvalidators: map[string]func(string) error{
			"first": func(string) error {
				called["first"] = true

				return nil
			},
			"second": func(string) error {
				called["second"] = true

				return nil
			},
		},
	}

	// Act
	err := value.Validate("привет")

	// Assert
	suite.Require().NoError(err)
	suite.Equal(map[string]bool{
		"first":  true,
		"second": true,
	}, called)
}

func (suite *BaseTestSuite) TestComplete() {
	for _, test := range []struct {
		name              string
		prepareError      error
		subvalidatorError error
		candidates        []string
		want              []cobra.Completion
		directive         cobra.ShellCompDirective
		calls             int
	}{
		{
			name:         "preparation failure falls back to completion",
			prepareError: errors.New("cannot prepare"),
			want:         []cobra.Completion{"привет"},
			directive:    cobra.ShellCompDirectiveNoSpace,
			calls:        1,
			candidates:   []string{"привет"},
		},
		{
			name:      "valid input returned unchanged",
			want:      []cobra.Completion{"привет"},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:              "constraint failure falls back with raw input",
			subvalidatorError: errors.New("constraint failed"),
			candidates:        []string{"привет"},
			want:              []cobra.Completion{"привет"},
			directive:         cobra.ShellCompDirectiveNoSpace,
			calls:             1,
		},
		{
			name:         "sort and deduplicate callback candidates",
			prepareError: errors.New("partial input"),
			candidates: []string{
				"привет2",
				"привет1",
				"привет2",
				"привет1",
			},
			want: []cobra.Completion{
				"привет1",
				"привет2",
			},
			directive: cobra.ShellCompDirectiveNoSpace,
			calls:     1,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			calls := 0
			prepareCalls := 0

			var raw, prepared, completionInput string

			value := baseValue[string]{
				toValue: func(input string) (string, error) {
					prepareCalls++
					raw = input

					return strings.ToUpper(input), test.prepareError
				},
				subvalidators: map[string]func(string) error{
					"constraint": func(input string) error {
						prepared = input

						return test.subvalidatorError
					},
				},
				toComplete: func(input string) ([]string, cobra.ShellCompDirective) {
					calls++
					completionInput = input

					return test.candidates, cobra.ShellCompDirectiveNoSpace
				},
			}

			// Act
			completions, directive := value.Complete("привет")

			// Assert
			suite.Equal(test.want, completions)
			suite.Equal(test.directive, directive)
			suite.Equal(test.calls, calls)
			suite.Equal(1, prepareCalls)
			suite.Equal("привет", raw)

			if test.prepareError == nil {
				suite.Equal("ПРИВЕТ", prepared)
			}

			if calls > 0 {
				suite.Equal("привет", completionInput)
			}
		})
	}
}

func (suite *BaseTestSuite) TestCompletionSortsWithKeepOrder() {
	// Arrange
	candidates := []string{
		"b",
		"a",
		"b",
	}
	value := baseValue[string]{
		toValue: func(string) (string, error) {
			return "", errors.New("partial input")
		},
		toComplete: func(string) ([]string, cobra.ShellCompDirective) {
			return candidates, cobra.ShellCompDirectiveKeepOrder
		},
	}

	// Act
	completed, directive := value.Complete("привет")

	// Assert
	suite.Equal([]cobra.Completion{
		"a",
		"b",
	}, completed)
	suite.Equal(cobra.ShellCompDirectiveKeepOrder, directive)
}

func (suite *BaseTestSuite) TestNewBase() {
	// Act
	value := newBase[string]("str", 2)

	// Assert
	suite.Equal("str()", value.String())
	suite.NotNil(value.subvalidators)
	suite.NotNil(value.subvalidatorDetails)
}

func (suite *BaseTestSuite) TestDefaultCompletion() {
	// Arrange
	value := newBase[string]("str", 1)
	value.toValue = func(input string) (string, error) {
		return input, nil
	}

	value.setSubvalidator("reject", "", func(string) error {
		return errors.New("invalid input")
	})

	// Act
	completions, directive := value.Complete("привет")

	// Assert
	suite.Nil(completions)
	suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
}

func (suite *BaseTestSuite) TestString() {
	for _, test := range []struct {
		name    string
		details map[string]string
		want    string
	}{
		{
			name: "no details",
			want: "str()",
		},
		{
			name: "empty detail uses name only",
			details: map[string]string{
				"re": "",
			},
			want: "str(re)",
		},
		{
			name: "sorted names with details",
			details: map[string]string{
				"re":         "^привет$",
				"min-length": "1",
				"max-length": "6",
			},
			want: "str(max-length:6, min-length:1, re:^привет$)",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value := newBase[string]("str", len(test.details))
			value.subvalidatorDetails = test.details

			// Act
			description := value.String()

			// Assert
			suite.Equal(test.want, description)
		})
	}
}

func (suite *BaseTestSuite) TestSetSubvalidatorReplacesDetailsAndValidation() {
	for _, detail := range []string{
		"^привет$",
		"",
	} {
		suite.Run(detail, func() {
			// Arrange
			value := newBase[string]("str", 1)
			value.toValue = func(input string) (string, error) { return input, nil }
			value.setSubvalidator("re", "old", func(string) error {
				return errors.New("old validator")
			})

			var received string

			// Act
			value.setSubvalidator("re", detail, func(input string) error {
				received = input

				return nil
			})
			err := value.Validate("привет")

			// Assert
			suite.Require().NoError(err)
			suite.Equal("привет", received)
			suite.Equal(detail, value.subvalidatorDetails["re"])
			suite.Len(value.subvalidators, 1)
		})
	}
}

func TestBase(t *testing.T) {
	t.Parallel()

	suite.Run(t, &BaseTestSuite{})
}
