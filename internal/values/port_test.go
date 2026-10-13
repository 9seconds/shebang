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

type PortTestSuite struct {
	suite.Suite
}

func (suite *PortTestSuite) TestParsing() {
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
			name:  "ordinary port",
			input: "8080",
		},
		{
			name:  "leading zeros remain decimal",
			input: "00080",
		},
		{
			name:  "maximum",
			input: strconv.FormatUint(math.MaxUint16, 10),
		},
		{
			name:  "overflow",
			input: strconv.FormatUint(math.MaxUint16+1, 10),
			want:  strconv.ErrRange,
		},
		{
			name: "empty",
			want: strconv.ErrSyntax,
		},
		{
			name:  "negative",
			input: "-1",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "plus sign",
			input: "+80",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "service name",
			input: "http",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "unicode",
			input: "привет",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "fraction",
			input: "80.0",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "whitespace",
			input: " 80 ",
			want:  strconv.ErrSyntax,
		},
		{
			name:  "hexadecimal",
			input: "0x50",
			want:  strconv.ErrSyntax,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.New("port", nil, nil)
			suite.Require().NoError(err)
			suite.Equal("port()", value.String())

			// Act
			err = value.Validate(test.input)

			// Assert
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

func (suite *PortTestSuite) TestRanges() {
	for _, test := range []struct {
		name     string
		category string
		first    uint16
		last     uint16
	}{
		{
			name:     "well-known",
			category: "well-known",
			last:     1023,
		},
		{
			name:     "registered",
			category: "registered",
			first:    1024,
			last:     49151,
		},
		{
			name:     "ephemeral",
			category: "ephemeral",
			first:    49152,
			last:     math.MaxUint16,
		},
	} {
		suite.Run(test.name, func() {
			for _, expected := range []bool{true, false} {
				// Arrange
				value, err := values.NewPort(nil, map[string][]any{
					test.name: {expected},
				})
				suite.Require().NoError(err)
				suite.Equal("port("+test.category+")", value.String())

				for _, port := range []uint16{0, 1023, 1024, 49151, 49152, math.MaxUint16} {
					// Act
					err := value.Validate(strconv.FormatUint(uint64(port), 10))

					// Assert
					inside := test.first <= port && port <= test.last
					if inside == expected {
						suite.Require().NoError(err)

						continue
					}

					var rangeError *values.ErrPortRange

					constructed := values.NewErrPortRange(port, test.category, expected)

					suite.Require().ErrorAs(err, &rangeError)
					suite.Equal(*constructed, *rangeError)

					message := "port " + strconv.FormatUint(uint64(port), 10) + " is "
					if expected {
						message += "not "
					}

					message += test.category
					suite.Require().EqualError(err, message)
				}
			}
		})
	}
}

func (suite *PortTestSuite) TestProperties() {
	for _, property := range []string{
		"well-known",
		"registered",
		"ephemeral",
	} {
		suite.Run(property, func() {
			for _, test := range []struct {
				name      string
				arguments []any
				want      error
			}{
				{
					name: "missing",
					want: utils.ErrEmpty,
				},
				{
					name:      "multiple",
					arguments: []any{true, false},
					want:      utils.ErrSingleArgumentExpected,
				},
				{
					name:      "string is not converted",
					arguments: []any{"true"},
				},
			} {
				suite.Run(test.name, func() {
					// Arrange
					properties := map[string][]any{
						property: test.arguments,
					}

					// Act
					value, err := values.NewPort(nil, properties)

					// Assert
					suite.Require().ErrorContains(err, "cannot add validator "+property+":")
					suite.Nil(value)

					if test.want != nil {
						suite.Require().ErrorIs(err, test.want)
					} else {
						var typeError *utils.ArgumentTypeError

						suite.Require().ErrorAs(err, &typeError)
						suite.Equal("bool", typeError.Expected)
					}
				})
			}
		})
	}

	// Arrange
	properties := map[string][]any{
		"unknown": {true},
	}

	// Act
	value, err := values.NewPort(nil, properties)

	// Assert
	suite.Require().ErrorIs(err, values.ErrUnknownSubvalidator)
	suite.Nil(value)
}

func (suite *PortTestSuite) TestCombinedConstraintsAndCompletion() {
	// Arrange
	value, err := values.NewPort(nil, map[string][]any{
		"well-known": {false},
		"ephemeral":  {false},
	})
	suite.Require().NoError(err)

	for _, test := range []struct {
		input string
		valid bool
	}{
		{
			input: "8080",
			valid: true,
		},
		{
			input: "80",
		},
		{
			input: "65535",
		},
	} {
		suite.Run(test.input, func() {
			// Act
			err := value.Validate(test.input)

			// Assert
			if test.valid {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}

	for _, input := range []string{
		"",
		"80",
		"привет",
	} {
		// Act
		candidates, directive := value.Complete(input)

		// Assert
		suite.Nil(candidates)
		suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
	}
}

func TestPort(t *testing.T) {
	t.Parallel()

	suite.Run(t, &PortTestSuite{})
}
