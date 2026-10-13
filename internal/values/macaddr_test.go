package values_test

import (
	"net"
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type MacAddrTestSuite struct {
	suite.Suite
}

func (suite *MacAddrTestSuite) TestAcceptedFormats() {
	for _, address := range []struct {
		name  string
		input string
	}{
		{
			name:  "MAC-48",
			input: "00:00:5e:00:53:01",
		},
		{
			name:  "EUI-64",
			input: "02:00:5e:10:00:00:00:01",
		},
		{
			name:  "InfiniBand",
			input: "00:00:00:00:fe:80:00:00:00:00:00:00:02:00:5e:10:00:00:00:01",
		},
	} {
		suite.Run(address.name, func() {
			compact := strings.ReplaceAll(address.input, ":", "")
			groups := make([]string, 0, len(compact)/4)

			for offset := 0; offset < len(compact); offset += 4 {
				groups = append(groups, compact[offset:offset+4])
			}

			for _, input := range []string{
				address.input,
				strings.ReplaceAll(address.input, ":", "-"),
				strings.Join(groups, "."),
				compact,
				strings.ToUpper(address.input),
			} {
				suite.Run(input, func() {
					// Arrange
					value, err := values.NewMacAddr(nil, nil)
					suite.Require().NoError(err)

					// Act
					err = value.Validate(input)
					candidates, directive := value.Complete(input)

					// Assert
					suite.Require().NoError(err)
					suite.Equal("mac-addr()", value.String())
					suite.Equal([]cobra.Completion{input}, candidates)
					suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
				})
			}
		})
	}
}

func (suite *MacAddrTestSuite) TestSpecialAddresses() {
	for _, input := range []string{
		"00:00:00:00:00:00",
		"ff:ff:ff:ff:ff:ff",
		"01:00:5e:00:00:01",
		"02:00:5e:00:53:01",
		"aA:Bb:cC:Dd:eE:Ff",
	} {
		suite.Run(input, func() {
			// Arrange
			value, err := values.NewMacAddr(nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(input)

			// Assert
			suite.Require().NoError(err)
		})
	}
}

func (suite *MacAddrTestSuite) TestInvalidInput() {
	for _, test := range []struct {
		name  string
		input string
	}{
		{
			name: "empty",
		},
		{
			name:  "partial",
			input: "00:00:5e",
		},
		{
			name:  "five bytes",
			input: "00:00:5e:00:53",
		},
		{
			name:  "seven bytes",
			input: "00:00:5e:00:53:01:02",
		},
		{
			name:  "nine bytes",
			input: "00:00:5e:00:53:01:02:03:04",
		},
		{
			name:  "nineteen bytes",
			input: strings.Repeat("00:", 18) + "00",
		},
		{
			name:  "twenty-one bytes",
			input: strings.Repeat("00:", 20) + "00",
		},
		{
			name:  "unsupported dotted length",
			input: "0000.5e00.5301.0203.0405",
		},
		{
			name:  "unsupported compact length",
			input: "00005e00530102",
		},
		{
			name:  "odd compact length",
			input: "00005e0053010",
		},
		{
			name:  "mixed separators",
			input: "00:00-5e:00:53:01",
		},
		{
			name:  "single digit octet",
			input: "0:00:5e:00:53:01",
		},
		{
			name:  "three digit octet",
			input: "000:00:5e:00:53:01",
		},
		{
			name:  "invalid colon hex",
			input: "00:00:5g:00:53:01",
		},
		{
			name:  "invalid dotted hex",
			input: "0000.5g00.5301",
		},
		{
			name:  "invalid second dotted byte",
			input: "0000.5e0g.5301",
		},
		{
			name:  "mixed dotted separators",
			input: "0000.5e00-5301",
		},
		{
			name:  "incomplete dotted group",
			input: "0000.5e00.530",
		},
		{
			name:  "invalid compact hex",
			input: "00005e00530g",
		},
		{
			name:  "leading space",
			input: " 00:00:5e:00:53:01",
		},
		{
			name:  "trailing newline",
			input: "00:00:5e:00:53:01\n",
		},
		{
			name:  "Unicode",
			input: "привет",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewMacAddr(nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			var addressError *net.AddrError

			suite.Require().ErrorAs(err, &addressError)
			suite.Equal("invalid MAC address", addressError.Err)
			suite.Equal(test.input, addressError.Addr)
			suite.Empty(candidates)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func (suite *MacAddrTestSuite) TestSubvalidatorsRejected() {
	for _, test := range []struct {
		name          string
		subvalidators map[string][]any
	}{
		{
			name: "unsupported subvalidator",
			subvalidators: map[string][]any{
				"length": {int64(6)},
			},
		},
		{
			name: "empty subvalidator",
			subvalidators: map[string][]any{
				"unknown": {},
			},
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			subvalidators := test.subvalidators

			// Act
			value, err := values.NewMacAddr(nil, subvalidators)
			factoryValue, factoryError := values.New("mac-addr", nil, subvalidators)

			// Assert
			suite.Require().ErrorIs(err, values.ErrNoSubvalidators)
			suite.Nil(value)

			suite.Require().ErrorIs(factoryError, values.ErrNoSubvalidators)
			suite.Nil(factoryValue)
		})
	}
}

func (suite *MacAddrTestSuite) TestEmptyConfiguration() {
	// Arrange
	properties := map[string]any{}
	subvalidators := map[string][]any{}

	// Act
	value, err := values.NewMacAddr(properties, subvalidators)

	// Assert
	suite.Require().NoError(err)
	suite.Require().NotNil(value)
	suite.Equal("mac-addr()", value.String())
	suite.Require().NoError(value.Validate("00:00:5e:00:53:01"))
}

func TestMacAddr(t *testing.T) {
	t.Parallel()

	suite.Run(t, &MacAddrTestSuite{})
}
