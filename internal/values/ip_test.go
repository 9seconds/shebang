package values_test

import (
	"net/netip"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type IPTestSuite struct {
	suite.Suite
}

func (suite *IPTestSuite) TestParsing() {
	for _, test := range []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "IPv4",
			input: "192.0.2.1",
			valid: true,
		},
		{
			name:  "IPv6",
			input: "2001:DB8::1",
			valid: true,
		},
		{
			name:  "mapped IPv4",
			input: "::ffff:192.0.2.1",
			valid: true,
		},
		{
			name:  "scoped IPv6",
			input: "fe80::1%eth0",
			valid: true,
		},
		{
			name: "empty",
		},
		{
			name:  "hostname",
			input: "localhost",
		},
		{
			name:  "unicode",
			input: "привет",
		},
		{
			name:  "out of range octet",
			input: "256.0.0.1",
		},
		{
			name:  "leading zero octet",
			input: "192.0.02.1",
		},
		{
			name:  "CIDR is not an address",
			input: "192.0.2.1/24",
		},
		{
			name:  "port is not an address",
			input: "192.0.2.1:80",
		},
		{
			name:  "bracketed IPv6",
			input: "[::1]",
		},
		{
			name:  "whitespace",
			input: " 192.0.2.1 ",
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.New("ip", nil)
			suite.Require().NoError(err)
			suite.Equal("ip(checks=classifiers)", value.String())

			err = value.Validate(test.input)
			if test.valid {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *IPTestSuite) TestTypes() {
	for _, test := range []struct {
		name     string
		accepted []string
		rejected []string
	}{
		{
			name:     "v4",
			accepted: []string{"192.0.2.1"},
			rejected: []string{"2001:db8::1", "::ffff:192.0.2.1"},
		},
		{
			name:     "v6",
			accepted: []string{"2001:db8::1", "::ffff:192.0.2.1"},
			rejected: []string{"192.0.2.1"},
		},
		{
			name:     "v4-in-v6",
			accepted: []string{"::ffff:192.0.2.1"},
			rejected: []string{"192.0.2.1", "2001:db8::1"},
		},
		{
			name:     "v6-only",
			accepted: []string{"2001:db8::1", "fe80::1%eth0"},
			rejected: []string{"192.0.2.1", "::ffff:192.0.2.1"},
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewIP(map[string][]any{
				"type": {test.name},
			})
			suite.Require().NoError(err)
			suite.Equal("ip(checks=classifiers, type:"+test.name+")", value.String())

			for _, address := range test.accepted {
				suite.Require().NoError(value.Validate(address))
			}

			for _, address := range test.rejected {
				var typeError *values.IPTypeError

				err := value.Validate(address)
				suite.Require().ErrorAs(err, &typeError)
				suite.Equal(*values.NewIPTypeError(netip.MustParseAddr(address), test.name), *typeError)
				suite.Require().ErrorContains(err, "is not of type "+test.name)
			}
		})
	}
}

func (suite *IPTestSuite) TestClassifiers() {
	for _, test := range []struct {
		name     string
		positive string
		negative string
	}{
		{
			name:     "global-unicast",
			positive: "192.0.2.1",
			negative: "127.0.0.1",
		},
		{
			name:     "iflocal-multicast",
			positive: "ff01::1",
			negative: "ff02::1",
		},
		{
			name:     "linklocal-unicast",
			positive: "fe80::1",
			negative: "2001:db8::1",
		},
		{
			name:     "linklocal-multicast",
			positive: "224.0.0.1",
			negative: "239.1.1.1",
		},
		{
			name:     "loopback",
			positive: "::1",
			negative: "2001:db8::1",
		},
		{
			name:     "multicast",
			positive: "ff02::1",
			negative: "2001:db8::1",
		},
		{
			name:     "private",
			positive: "10.1.2.3",
			negative: "192.0.2.1",
		},
		{
			name:     "unspecified",
			positive: "0.0.0.0",
			negative: "192.0.2.1",
		},
	} {
		suite.Run(test.name, func() {
			for _, expected := range []bool{true, false} {
				value, err := values.NewIP(map[string][]any{
					test.name: {expected},
				})
				suite.Require().NoError(err)

				accepted, rejected := test.positive, test.negative
				if !expected {
					accepted, rejected = rejected, accepted
				}

				suite.Require().NoError(value.Validate(accepted))

				var classifierError *values.IPClassifierError

				err = value.Validate(rejected)
				suite.Require().ErrorAs(err, &classifierError)
				suite.Equal(*values.NewIPClassifierError(netip.MustParseAddr(rejected),
					test.name, expected), *classifierError)
				suite.Equal(classifierError.Error(), err.Error())
			}
		})
	}
}

func (suite *IPTestSuite) TestSubnets() {
	for _, test := range []struct {
		name     string
		subnets  []any
		accepted []string
		rejected []string
	}{
		{
			name:     "multiple families and boundaries",
			subnets:  []any{"192.0.2.19/24", "2001:db8::/32"},
			accepted: []string{"192.0.2.0", "192.0.2.255", "2001:db8:ffff::1"},
			rejected: []string{"192.0.3.0", "2001:db9::1", "::ffff:192.0.2.1"},
		},
		{
			name:     "mapped prefix is not IPv4",
			subnets:  []any{"::ffff:192.0.2.0/120"},
			accepted: []string{"::ffff:192.0.2.1"},
			rejected: []string{"192.0.2.1"},
		},
		{
			name:     "zone does not match CIDR",
			subnets:  []any{"fe80::/10"},
			accepted: []string{"fe80::1"},
			rejected: []string{"fe80::1%eth0"},
		},
		{
			name:     "single address subnet",
			subnets:  []any{"192.0.2.1/32"},
			accepted: []string{"192.0.2.1"},
			rejected: []string{"192.0.2.2"},
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewIP(map[string][]any{
				"subnets": test.subnets,
			})
			suite.Require().NoError(err)

			for _, address := range test.accepted {
				suite.Require().NoError(value.Validate(address))
			}

			for _, address := range test.rejected {
				var subnetError *values.IPSubnetError

				err := value.Validate(address)
				suite.Require().ErrorAs(err, &subnetError)
				suite.Equal(netip.MustParseAddr(address), subnetError.Address)
				suite.Require().Len(subnetError.Subnets, len(test.subnets))
				suite.Require().ErrorContains(err, "does not belong to any configured subnet")
			}
		})
	}
}

func (suite *IPTestSuite) TestInvalidProperties() {
	for _, test := range []struct {
		name      string
		property  string
		arguments []any
		want      error
		message   string
	}{
		{
			name:     "unknown property",
			property: "unknown",
			want:     values.ErrUnknownProperty,
		},
		{
			name:     "missing type",
			property: "type",
			want:     utils.ErrEmpty,
		},
		{
			name:      "multiple types",
			property:  "type",
			arguments: []any{"v4", "v6"},
			want:      utils.ErrSingleArgumentExpected,
		},
		{
			name:      "unknown type",
			property:  "type",
			arguments: []any{"V4"},
			message:   "unknown IP type V4",
		},
		{
			name:      "type is not a string",
			property:  "type",
			arguments: []any{true},
			message:   "expected string parameter, got bool",
		},
		{
			name:     "empty subnet list",
			property: "subnets",
			want:     utils.ErrEmpty,
		},
		{
			name:      "invalid prefix",
			property:  "subnets",
			arguments: []any{"192.0.2.0/33"},
			message:   "incorrect prefix 192.0.2.0/33:",
		},
		{
			name:      "subnet is not a string",
			property:  "subnets",
			arguments: []any{true},
			message:   "expected string parameter, got bool",
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewIP(map[string][]any{
				test.property: test.arguments,
			})
			suite.Require().ErrorContains(err, "cannot add validator "+test.property+":")
			suite.Nil(value)

			if test.want != nil {
				suite.Require().ErrorIs(err, test.want)
			} else {
				suite.Require().ErrorContains(err, test.message)
			}

			if test.property == "type" && test.name == "unknown type" {
				var typeError *values.IPTypeError

				suite.Require().ErrorAs(err, &typeError)
				suite.False(typeError.Address.IsValid())
				suite.Equal("V4", typeError.Type)
			}
		})
	}
}

func (suite *IPTestSuite) TestClassifierArguments() {
	for _, property := range []string{
		"global-unicast", "iflocal-multicast", "linklocal-unicast", "linklocal-multicast",
		"loopback", "multicast", "private", "unspecified",
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
					name:      "string is not a boolean",
					arguments: []any{"true"},
				},
			} {
				suite.Run(test.name, func() {
					value, err := values.NewIP(map[string][]any{
						property: test.arguments,
					})
					suite.Require().Error(err)
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
}

func (suite *IPTestSuite) TestCombinedConstraints() {
	value, err := values.NewIP(map[string][]any{
		"type":     {"v4"},
		"private":  {true},
		"loopback": {false},
		"subnets":  {"10.0.0.0/8", "192.168.0.0/16"},
	})
	suite.Require().NoError(err)
	suite.Equal("ip(checks=classifiers, subnets:10.0.0.0/8,192.168.0.0/16, type:v4)",
		value.String())
	suite.Require().NoError(value.Validate("10.1.2.3"))
	suite.Require().NoError(value.Validate("192.168.1.10"))
	suite.Require().Error(value.Validate("172.16.1.10"))
	suite.Require().Error(value.Validate("192.0.2.1"))
	suite.Require().Error(value.Validate("::ffff:10.1.2.3"))
}

func (suite *IPTestSuite) TestSubnetErrorCopiesPrefixes() {
	prefixes := []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}
	address := netip.MustParseAddr("198.51.100.1")
	err := values.NewIPSubnetError(address, prefixes)
	prefixes[0] = netip.MustParsePrefix("198.51.100.0/24")

	suite.Equal(address, err.Address)
	suite.Equal([]netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}, err.Subnets)
	suite.Equal("IP address 198.51.100.1 does not belong to any configured subnet", err.Error())
}

func (suite *IPTestSuite) TestEveryClassifierIsChecked() {
	value, err := values.NewIP(map[string][]any{
		"private":   {true},
		"loopback":  {false},
		"multicast": {false},
	})
	suite.Require().NoError(err)
	suite.Require().NoError(value.Validate("10.1.2.3"))

	// Repeat to exercise map iteration orders with both matching and failing checks.
	for _, address := range []string{"192.0.2.1", "127.0.0.1", "239.1.1.1"} {
		suite.Run(address, func() {
			for range 32 {
				var classifierError *values.IPClassifierError

				suite.Require().ErrorAs(value.Validate(address), &classifierError)
			}
		})
	}
}

func (suite *IPTestSuite) TestCompletion() {
	for _, test := range []struct {
		name       string
		properties map[string][]any
		input      string
		want       []cobra.Completion
		directive  cobra.ShellCompDirective
	}{
		{
			name: "family filter without classifiers",
			properties: map[string][]any{
				"subnets": {"192.0.2.0/24", "2001:db8::/32"},
				"type":    {"v6-only"},
			},
			want:      []cobra.Completion{"2001:0db8:0000:0000:0000:0000:0000:000", "2001:db8:"},
			directive: cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:      "no configured candidates",
			want:      []cobra.Completion{},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "IPv4 prefix suggestions are deduplicated",
			properties: map[string][]any{
				"subnets": {"192.0.2.1/32", "192.0.2.1/32", "192.0.2.19/24"},
			},
			input:     "192.0.2.",
			want:      []cobra.Completion{"192.0.2."},
			directive: cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "IPv6 compressed and expanded forms",
			properties: map[string][]any{
				"subnets": {"2001:db8::/32"},
			},
			want:      []cobra.Completion{"2001:0db8:0000:0000:0000:0000:0000:000", "2001:db8:"},
			directive: cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "type and classifier filter candidates",
			properties: map[string][]any{
				"subnets":  {"127.0.0.0/8", "192.0.2.0/24", "2001:db8::/32"},
				"type":     {"v4"},
				"loopback": {false},
			},
			want:      []cobra.Completion{"192.0.2."},
			directive: cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "partial input with no match",
			properties: map[string][]any{
				"subnets": {"192.0.2.0/24"},
			},
			input:     "привет",
			want:      []cobra.Completion{},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:      "valid original IPv6 spelling returned unchanged",
			input:     "2001:DB8::1",
			want:      []cobra.Completion{"2001:DB8::1"},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "valid base address takes precedence over prefix suggestions",
			properties: map[string][]any{
				"subnets": {"192.0.2.0/24"},
			},
			input:     "192.0.2.0",
			want:      []cobra.Completion{"192.0.2.0"},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewIP(test.properties)
			suite.Require().NoError(err)

			candidates, directive := value.Complete(test.input)
			suite.Equal(test.want, candidates)
			suite.Equal(test.directive, directive)
		})
	}
}

func TestIP(t *testing.T) {
	t.Parallel()

	suite.Run(t, &IPTestSuite{})
}
