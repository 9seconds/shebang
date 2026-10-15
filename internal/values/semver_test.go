package values_test

import (
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type SemverTestSuite struct {
	suite.Suite
}

func (suite *SemverTestSuite) TestParsing() {
	for _, input := range []string{
		"1.2.3",
		"1",
		"v1.2",
		"v01.002.0003",
		"1.2.3-beta.1+build.123",
		"0.0.0",
		"18446744073709551615.0.0",
		"1.2.3+" + strings.Repeat("a", semver.MaxVersionLen-6),
	} {
		suite.Run(input, func() {
			// Arrange
			value, err := values.NewSemver(nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(input)
			candidates, directive := value.Complete(input)

			// Assert
			suite.Require().NoError(err)
			suite.Equal("semver()", value.String())
			suite.Equal([]cobra.Completion{input}, candidates)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func (suite *SemverTestSuite) TestInvalidVersions() {
	for _, test := range []struct {
		name  string
		input string
		want  error
	}{
		{
			name: "empty",
		},
		{
			name:  "uppercase prefix",
			input: "V1.2.3",
		},
		{
			name:  "negative segment",
			input: "-1.2.3",
		},
		{
			name:  "too many segments",
			input: "1.2.3.4",
		},
		{
			name:  "missing minor",
			input: "1..3",
		},
		{
			name:  "empty prerelease",
			input: "1.2.3-",
		},
		{
			name:  "empty metadata",
			input: "1.2.3+",
		},
		{
			name:  "numeric prerelease leading zeros",
			input: "1.2.3-01",
			want:  semver.ErrSegmentStartsZero,
		},
		{
			name:  "empty prerelease identifier",
			input: "1.2.3-alpha..1",
		},
		{
			name:  "whitespace",
			input: " 1.2.3 ",
		},
		{
			name:  "Unicode",
			input: "1.2.3-привет",
		},
		{
			name:  "segment overflow",
			input: "18446744073709551616.0.0",
		},
		{
			name:  "length exceeds maximum",
			input: "1.2.3+" + strings.Repeat("a", semver.MaxVersionLen-5),
			want:  semver.ErrVersionTooLong,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewSemver(nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			suite.Require().Error(err)
			suite.Empty(candidates)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if test.want != nil {
				suite.Require().ErrorIs(err, test.want)
			}
		})
	}
}

func (suite *SemverTestSuite) TestConstraints() {
	for _, test := range []struct {
		constraint string
		accepted   []string
		rejected   []string
	}{
		{
			constraint: ">= 1.2.3, < 2.0.0",
			accepted: []string{
				"v01.02.003",
				"1.9.9",
			},
			rejected: []string{
				"1.2.2",
				"2.0.0",
				"1.3.0-beta.1",
			},
		},
		{
			constraint: ">= 1.2.3 < 2.0.0",
			accepted:   []string{"1.5.0"},
			rejected:   []string{"2.0.0"},
		},
		{
			constraint: "= 1.2.3",
			accepted:   []string{"1.2.3+build.1"},
			rejected:   []string{"1.2.4"},
		},
		{
			constraint: "!= 1.2.3",
			accepted:   []string{"1.2.4"},
			rejected:   []string{"1.2.3"},
		},
		{
			constraint: "> 1.2.3, <= 1.2.5",
			accepted:   []string{"1.2.5"},
			rejected: []string{
				"1.2.3",
				"1.2.6",
			},
		},
		{
			constraint: "1.2.x",
			accepted:   []string{"1.2.9"},
			rejected:   []string{"1.3.0"},
		},
		{
			constraint: "~1.2.3",
			accepted:   []string{"1.2.9"},
			rejected: []string{
				"1.2.2",
				"1.3.0",
			},
		},
		{
			constraint: "^1.2.3",
			accepted:   []string{"1.9.0"},
			rejected:   []string{"2.0.0"},
		},
		{
			constraint: "^0.2.3",
			accepted:   []string{"0.2.9"},
			rejected:   []string{"0.3.0"},
		},
		{
			constraint: "^0.0.3",
			accepted:   []string{"0.0.3"},
			rejected:   []string{"0.0.4"},
		},
		{
			constraint: "1.2.3 - 2.0.0",
			accepted: []string{
				"1.2.3",
				"2.0.0",
			},
			rejected: []string{"2.0.1"},
		},
		{
			constraint: "< 1.0.0 || >= 2.0.0",
			accepted: []string{
				"0.9.0",
				"2.0.0",
			},
			rejected: []string{"1.5.0"},
		},
		{
			constraint: ">= 1.2.3-0",
			accepted:   []string{"1.2.3-beta.1"},
			rejected:   []string{"1.2.2"},
		},
		{
			constraint: ">= 2.0.0, < 1.0.0",
			rejected:   []string{"1.5.0"},
		},
	} {
		suite.Run(test.constraint, func() {
			for _, valid := range []bool{true, false} {
				inputs := test.rejected
				if valid {
					inputs = test.accepted
				}

				for _, input := range inputs {
					suite.Run(input, func() {
						// Arrange
						value, err := values.New("semver", nil, map[string][]any{
							"constraint": {test.constraint},
						})
						suite.Require().NoError(err)

						// Act
						err = value.Validate(input)
						candidates, directive := value.Complete(input)

						// Assert
						suite.Equal("semver(constraint:"+test.constraint+")", value.String())
						suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

						if valid {
							suite.Require().NoError(err)
							suite.Equal([]cobra.Completion{input}, candidates)
						} else {
							suite.Require().Error(err)
							suite.NotEmpty(err.Error())
							suite.Empty(candidates)
						}
					})
				}
			}
		})
	}
}

func (suite *SemverTestSuite) TestConfigurationErrors() {
	for _, test := range []struct {
		name string
		args []any
		want error
	}{
		{
			name: "missing argument",
			want: utils.ErrEmpty,
		},
		{
			name: "multiple arguments",
			args: []any{
				">= 1",
				"< 2",
			},
			want: utils.ErrSingleArgumentExpected,
		},
		{
			name: "wrong type",
			args: []any{true},
		},
		{
			name: "null",
			args: []any{nil},
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			subvalidators := map[string][]any{
				"constraint": test.args,
			}

			// Act
			value, err := values.NewSemver(nil, subvalidators)

			// Assert
			suite.Nil(value)
			suite.Require().ErrorContains(err, "incorrect subvalidator constraint")

			if test.want != nil {
				suite.Require().ErrorIs(err, test.want)
			} else {
				var typeError *utils.ArgumentTypeError

				suite.Require().ErrorAs(err, &typeError)
				suite.Equal("string", typeError.Expected)
			}
		})
	}

	for _, constraint := range []string{
		"привет",
		">>> 1.2.3",
		">= 1.0.0 garbage",
	} {
		suite.Run(constraint, func() {
			// Arrange
			subvalidators := map[string][]any{
				"constraint": {constraint},
			}

			// Act
			value, err := values.New("semver", nil, subvalidators)

			// Assert
			suite.Nil(value)
			suite.Require().ErrorContains(err, "incorrect constraint:")
		})
	}
}

func (suite *SemverTestSuite) TestUnknownSubvalidator() {
	// Arrange
	subvalidators := map[string][]any{
		"unknown": {},
	}

	// Act
	value, err := values.New("semver", nil, subvalidators)

	// Assert
	suite.Nil(value)
	suite.Require().ErrorIs(err, values.ErrUnknownSubvalidator)
}

func (suite *SemverTestSuite) TestJoinedDiagnostics() {
	// Arrange
	value, err := values.NewSemver(nil, map[string][]any{
		"constraint": {">= 2.0.0, < 1.0.0"},
	})
	suite.Require().NoError(err)

	// Act
	err = value.Validate("1.5.0")

	// Assert
	suite.Require().EqualError(err,
		"\"1.5.0\" is less than \"2.0.0\"\n\"1.5.0\" is greater than or equal to \"1.0.0\"")
}

func TestSemver(t *testing.T) {
	t.Parallel()

	suite.Run(t, &SemverTestSuite{})
}
