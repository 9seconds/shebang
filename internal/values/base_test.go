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
		name         string
		prepareError error
		checkError   error
		want         error
		checksCalled int
	}{
		{
			name:         "preparation error skips checks",
			prepareError: want,
			want:         want,
		},
		{
			name:         "check error propagated",
			checkError:   want,
			want:         want,
			checksCalled: 1,
		},
		{
			name:         "prepared value passes checks",
			checksCalled: 1,
		},
	} {
		suite.Run(test.name, func() {
			prepareCalls := 0
			checkCalls := 0
			value := baseValue[string]{
				prepare: func(input string) (string, error) {
					prepareCalls++

					suite.Equal("привет", input)

					return strings.ToUpper(input), test.prepareError
				},
				checks: map[string]func(string) error{
					"check": func(input string) error {
						checkCalls++

						suite.Equal("ПРИВЕТ", input)

						return test.checkError
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
			suite.Equal(test.checksCalled, checkCalls)
		})
	}
}

func (suite *BaseTestSuite) TestAllChecksRun() {
	called := make(map[string]bool)
	value := baseValue[string]{
		prepare: func(input string) (string, error) {
			return input, nil
		},
		checks: map[string]func(string) error{
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
		name         string
		prepareError error
		checkError   error
		candidates   []string
		want         []cobra.Completion
		directive    cobra.ShellCompDirective
		calls        int
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
			name:       "constraint failure falls back with raw input",
			checkError: errors.New("constraint failed"),
			candidates: []string{"привет"},
			want:       []cobra.Completion{"привет"},
			directive:  cobra.ShellCompDirectiveNoSpace,
			calls:      1,
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
				prepare: func(input string) (string, error) {
					prepareCalls++

					suite.Equal("привет", input)

					return strings.ToUpper(input), test.prepareError
				},
				checks: map[string]func(string) error{
					"constraint": func(input string) error {
						suite.Equal("ПРИВЕТ", input)

						return test.checkError
					},
				},
				complete: func(input string) ([]string, cobra.ShellCompDirective) {
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

func (suite *BaseTestSuite) TestCompletionDoesNotMutateSharedCandidates() {
	candidates := []string{"b", "a", "b"}
	value := baseValue[string]{
		prepare: func(string) (string, error) {
			return "", errors.New("partial input")
		},
		complete: func(string) ([]string, cobra.ShellCompDirective) {
			return candidates, cobra.ShellCompDirectiveKeepOrder
		},
	}
	completed, directive := value.Complete("привет")
	suite.Equal([]cobra.Completion{"b", "a"}, completed)
	suite.Equal(cobra.ShellCompDirectiveKeepOrder, directive)
	suite.Equal([]string{"b", "a", "b"}, candidates)
}

func TestBase(t *testing.T) {
	t.Parallel()

	suite.Run(t, &BaseTestSuite{})
}
