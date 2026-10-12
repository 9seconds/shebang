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
			prepareCalls := 0
			subvalidatorCalls := 0
			value := baseValue[string]{
				toValue: func(input string) (string, error) {
					prepareCalls++

					suite.Equal("привет", input)

					return strings.ToUpper(input), test.prepareError
				},
				subvalidators: map[string]func(string) error{
					"subvalidator": func(input string) error {
						subvalidatorCalls++

						suite.Equal("ПРИВЕТ", input)

						return test.subvalidatorError
					},
				},
			}

			err := value.Validate("привет")
			if test.want != nil {
				suite.Require().ErrorIs(err, test.want)
			} else {
				suite.Require().NoError(err)
			}

			suite.Equal(1, prepareCalls)
			suite.Equal(test.subvalidatorsCalled, subvalidatorCalls)
		})
	}
}

func (suite *BaseTestSuite) TestAllSubvalidatorsRun() {
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
	suite.Require().NoError(value.Validate("привет"))
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
			candidates:   []string{"привет2", "привет1", "привет2", "привет1"},
			want:         []cobra.Completion{"привет1", "привет2"},
			directive:    cobra.ShellCompDirectiveNoSpace,
			calls:        1,
		},
	} {
		suite.Run(test.name, func() {
			calls := 0
			prepareCalls := 0
			value := baseValue[string]{
				toValue: func(input string) (string, error) {
					prepareCalls++

					suite.Equal("привет", input)

					return strings.ToUpper(input), test.prepareError
				},
				subvalidators: map[string]func(string) error{
					"constraint": func(input string) error {
						suite.Equal("ПРИВЕТ", input)

						return test.subvalidatorError
					},
				},
				toComplete: func(input string) ([]string, cobra.ShellCompDirective) {
					calls++

					suite.Equal("привет", input)

					return test.candidates, cobra.ShellCompDirectiveNoSpace
				},
			}
			completions, directive := value.Complete("привет")
			suite.Equal(test.want, completions)
			suite.Equal(test.directive, directive)
			suite.Equal(test.calls, calls)
			suite.Equal(1, prepareCalls)
		})
	}
}

func (suite *BaseTestSuite) TestCompletionSortsWithKeepOrder() {
	candidates := []string{"b", "a", "b"}
	value := baseValue[string]{
		toValue: func(string) (string, error) {
			return "", errors.New("partial input")
		},
		toComplete: func(string) ([]string, cobra.ShellCompDirective) {
			return candidates, cobra.ShellCompDirectiveKeepOrder
		},
	}
	completed, directive := value.Complete("привет")
	suite.Equal([]cobra.Completion{"a", "b"}, completed)
	suite.Equal(cobra.ShellCompDirectiveKeepOrder, directive)
}

func (suite *BaseTestSuite) TestNewBase() {
	value := newBase[string]("str", 2)
	suite.Equal("str()", value.String())
	suite.NotNil(value.subvalidators)
	suite.NotNil(value.subvalidatorDetails)
	value.toValue = func(input string) (string, error) {
		return input, nil
	}
	suite.Require().NoError(value.Validate("привет"))

	value.setSubvalidator("reject", "", func(string) error {
		return errors.New("invalid input")
	})
	completions, directive := value.Complete("привет")
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
			value := newBase[string]("str", len(test.details))
			value.subvalidatorDetails = test.details
			suite.Equal(test.want, value.String())
		})
	}
}

func (suite *BaseTestSuite) TestSetSubvalidatorReplacesDetailsAndValidation() {
	value := newBase[string]("str", 1)
	value.toValue = func(input string) (string, error) {
		return input, nil
	}
	want := errors.New("invalid input")

	value.setSubvalidator("re", "old", func(string) error {
		return want
	})
	suite.Equal("str(re:old)", value.String())
	suite.Require().ErrorIs(value.Validate("привет"), want)

	value.setSubvalidator("re", "^привет$", func(input string) error {
		suite.Equal("привет", input)

		return nil
	})
	suite.Equal("str(re:^привет$)", value.String())
	suite.Require().NoError(value.Validate("привет"))

	value.setSubvalidator("re", "", func(string) error {
		return nil
	})
	suite.Equal("str(re)", value.String())
}

func TestBase(t *testing.T) {
	t.Parallel()

	suite.Run(t, &BaseTestSuite{})
}
