package values_test

import (
	"net/netip"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type SubnetTestSuite struct {
	suite.Suite
}

func (suite *SubnetTestSuite) TestParsing() {
	for _, test := range []struct {
		input string
		valid bool
	}{
		{
			input: "192.0.2.19/24",
			valid: true,
		},
		{
			input: "0.0.0.0/0",
			valid: true,
		},
		{
			input: "255.255.255.255/32",
			valid: true,
		},
		{
			input: "::/0",
			valid: true,
		},
		{
			input: "2001:DB8::1/128",
			valid: true,
		},
		{
			input: "::ffff:192.0.2.1/120",
			valid: true,
		},
		{},
		{
			input: "192.0.2.1",
		},
		{
			input: "192.0.2.0/-1",
		},
		{
			input: "192.0.2.0/33",
		},
		{
			input: "2001:db8::/129",
		},
		{
			input: "fe80::1%eth0/64",
		},
		{
			input: "192.0.2.0/255.255.255.0",
		},
		{
			input: " 192.0.2.0/24",
		},
		{
			input: "привет/24",
		},
	} {
		suite.Run(test.input, func() {
			// Arrange
			value, err := values.NewSubnet(nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			suite.Equal("subnet()", value.String())
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if test.valid {
				suite.Require().NoError(err)
				suite.Equal([]cobra.Completion{test.input}, candidates)
			} else {
				suite.Require().Error(err)
				suite.Empty(candidates)
			}
		})
	}
}

func (suite *SubnetTestSuite) TestTypes() {
	for _, family := range []string{
		"v4",
		"v6",
		"v4-in-v6",
		"v6-only",
	} {
		for _, test := range []struct {
			input  string
			v4     bool
			mapped bool
		}{
			{
				input: "192.0.2.1/24",
				v4:    true,
			},
			{
				input: "2001:db8::1/32",
			},
			{
				input:  "::ffff:192.0.2.1/120",
				mapped: true,
			},
		} {
			suite.Run(family+"/"+test.input, func() {
				// Arrange
				families := map[string]bool{
					"v4":       test.v4,
					"v6":       !test.v4,
					"v4-in-v6": test.mapped,
					"v6-only":  !test.v4 && !test.mapped,
				}
				valid := families[family]
				value, err := values.New("subnet", nil, map[string][]any{
					"type": {family},
				})
				suite.Require().NoError(err)

				// Act
				err = value.Validate(test.input)

				// Assert
				if valid {
					suite.Require().NoError(err)
				} else {
					var typeError *values.ErrAddressType

					suite.Require().ErrorAs(err, &typeError)
					suite.Equal(family, typeError.Type)
					suite.Equal(netip.MustParsePrefix(test.input).Addr(), typeError.Address)
				}
			})
		}
	}
}

func (suite *SubnetTestSuite) TestOverlap() {
	for _, name := range []string{
		"overlaps",
		"not-overlaps",
	} {
		for _, test := range []struct {
			input   string
			overlap bool
		}{
			{
				input:   "10.0.0.0/8",
				overlap: true,
			},
			{
				input:   "10.2.3.4/24",
				overlap: true,
			},
			{
				input:   "0.0.0.0/0",
				overlap: true,
			},
			{
				input:   "192.168.1.0/24",
				overlap: true,
			},
			{
				input: "11.0.0.0/8",
			},
			{
				input:   "2001:db8:1::/48",
				overlap: true,
			},
			{
				input: "2001:db9::/32",
			},
			{
				input: "::ffff:10.0.0.0/104",
			},
		} {
			suite.Run(name+"/"+test.input, func() {
				// Arrange
				value, err := values.NewSubnet(nil, map[string][]any{
					name: {
						"192.168.0.1/16",
						"10.0.0.1/8",
						"2001:db8::1/32",
					},
				})
				suite.Require().NoError(err)

				// Act
				err = value.Validate(test.input)

				// Assert
				if test.overlap == (name == "overlaps") {
					suite.Require().NoError(err)
				} else {
					var overlapError *values.ErrSubnetOverlap

					suite.Require().ErrorAs(err, &overlapError)
					suite.Equal(netip.MustParsePrefix(test.input), overlapError.Subnet)
					//nolint:testifylint // Expected is the actual error field being verified.
					suite.Equal(name == "overlaps", overlapError.Expected)
					suite.NotEmpty(overlapError.Prefixes)
					suite.Contains(overlapError.Error(), "subnet "+test.input)
				}
			})
		}
	}
}

func (suite *SubnetTestSuite) TestCompletion() {
	for _, test := range []struct {
		name          string
		subvalidators map[string][]any
		input         string
		want          []cobra.Completion
	}{
		{
			name: "sorted and deduplicated masked addresses",
			subvalidators: map[string][]any{
				"overlaps": {
					"192.0.2.19/24",
					"10.0.0.1/8",
					"192.0.2.0/24",
				},
			},
			want: []cobra.Completion{
				"10.0.0.0/",
				"192.0.2.0/",
			},
		},
		{
			name: "IPv6 compressed and expanded",
			subvalidators: map[string][]any{
				"overlaps": {"2001:db8::1/32"},
			},
			input: "2001:",
			want: []cobra.Completion{
				"2001:0db8:0000:0000:0000:0000:0000:0000/",
				"2001:db8::/",
			},
		},
		{
			name: "completion extends slash input",
			subvalidators: map[string][]any{
				"overlaps": {"2001:db8::/32"},
			},
			input: "2001:db8::/",
			want:  []cobra.Completion{"2001:db8::/"},
		},
		{
			name: "type filters suggestions",
			subvalidators: map[string][]any{
				"type": {"v4"},
				"overlaps": {
					"10.0.0.0/8",
					"2001:db8::/32",
				},
			},
			want: []cobra.Completion{"10.0.0.0/"},
		},
		{
			name: "nonmatching input",
			subvalidators: map[string][]any{
				"overlaps": {"10.0.0.0/8"},
			},
			input: "192.",
		},
		{
			name: "exclusions disable generated suggestions",
			subvalidators: map[string][]any{
				"overlaps":     {"10.0.0.0/8"},
				"not-overlaps": {"10.1.0.0/16"},
			},
		},
		{
			name:  "no overlaps no suggestions",
			input: "10.",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewSubnet(nil, test.subvalidators)
			suite.Require().NoError(err)

			// Act
			candidates, directive := value.Complete(test.input)

			// Assert
			if len(test.want) > 0 {
				suite.Equal(test.want, candidates)
				suite.Equal(cobra.ShellCompDirectiveNoSpace, directive)
			} else {
				suite.Empty(candidates)
				suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
			}
		})
	}
}

func (suite *SubnetTestSuite) TestCombinedConstraints() {
	for _, input := range []string{
		"10.1.0.0/16",
		"10.0.0.0/8",
		"192.0.2.0/24",
		"2001:db8::/32",
		"10.2.3.4/24",
	} {
		suite.Run(input, func() {
			// Arrange
			value, err := values.NewSubnet(nil, map[string][]any{
				"type":         {"v4"},
				"overlaps":     {"10.0.0.1/8"},
				"not-overlaps": {"10.1.0.1/16"},
			})
			suite.Require().NoError(err)

			// Act
			err = value.Validate(input)
			candidates, directive := value.Complete(input)

			// Assert
			suite.Equal("subnet(not-overlaps:10.1.0.1/16, overlaps:10.0.0.1/8, type:v4)", value.String())
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if input == "10.2.3.4/24" {
				suite.Require().NoError(err)
				suite.Equal([]cobra.Completion{input}, candidates)
			} else {
				suite.Require().Error(err)
				suite.Empty(candidates)
			}
		})
	}
}

func (suite *SubnetTestSuite) TestConfigurationErrors() {
	for _, name := range []string{
		"type",
		"overlaps",
		"not-overlaps",
	} {
		for _, test := range []struct {
			name string
			args []any
			want error
		}{
			{
				name: "empty arguments",
				want: utils.ErrEmpty,
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
			suite.Run(name+"/"+test.name, func() {
				// Arrange
				subvalidators := map[string][]any{
					name: test.args,
				}

				// Act
				value, err := values.New("subnet", nil, subvalidators)

				// Assert
				suite.Nil(value)
				suite.Require().ErrorContains(err, "incorrect subvalidator "+name)

				if test.want != nil {
					suite.Require().ErrorIs(err, test.want)
				} else {
					var typeError *utils.ArgumentTypeError

					suite.Require().ErrorAs(err, &typeError)
				}
			})
		}
	}
}

func (suite *SubnetTestSuite) TestInvalidDeclarations() {
	for _, test := range []struct {
		name string
		args []any
		want error
	}{
		{
			name: "type",
			args: []any{
				"v4",
				"v6",
			},
			want: utils.ErrSingleArgumentExpected,
		},
		{
			name: "type",
			args: []any{"unknown"},
		},
		{
			name: "unknown",
			want: values.ErrUnknownSubvalidator,
		},
		{
			name: "overlaps",
			args: []any{"привет/24"},
		},
		{
			name: "not-overlaps",
			args: []any{"192.0.2.0/33"},
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			subvalidators := map[string][]any{
				test.name: test.args,
			}

			// Act
			value, err := values.NewSubnet(nil, subvalidators)

			// Assert
			suite.Nil(value)
			suite.Require().Error(err)

			switch {
			case test.want != nil:
				suite.Require().ErrorIs(err, test.want)
			case test.name == "type":
				var typeError *values.ErrAddressType

				suite.Require().ErrorAs(err, &typeError)
				suite.False(typeError.Address.IsValid())
			default:
				suite.ErrorContains(err, "incorrect prefix")
			}
		})
	}
}

func (suite *SubnetTestSuite) TestErrorOwnsPrefixes() {
	// Arrange
	prefix := netip.MustParsePrefix("10.0.0.0/8")
	prefixes := []netip.Prefix{prefix}
	input := netip.MustParsePrefix("192.0.2.0/24")

	// Act
	err := values.NewErrSubnetOverlap(input, prefixes, true)
	prefixes[0] = input

	// Assert
	suite.Equal([]netip.Prefix{prefix}, err.Prefixes)
	suite.Equal(input, err.Subnet)
	suite.Equal("subnet 192.0.2.0/24 must overlap with at least one subnet: [10.0.0.0/8]", err.Error())
}

func TestSubnet(t *testing.T) {
	t.Parallel()

	suite.Run(t, &SubnetTestSuite{})
}
