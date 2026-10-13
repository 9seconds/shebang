package values_test

import (
	"testing"

	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type UUIDTestSuite struct {
	suite.Suite
}

func (suite *UUIDTestSuite) TestValidateAndComplete() {
	for _, test := range []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "canonical version one",
			input: "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
			valid: true,
		},
		{
			name:  "uppercase hex",
			input: "F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6",
			valid: true,
		},
		{
			name:  "mixed case hex",
			input: "f81D4faE-7dEC-11d0-a765-00A0c91E6bF6",
			valid: true,
		},
		{
			name:  "compact hex",
			input: "f81d4fae7dec11d0a76500a0c91e6bf6",
			valid: true,
		},
		{
			name:  "braced",
			input: "{f81d4fae-7dec-11d0-a765-00a0c91e6bf6}",
			valid: true,
		},
		{
			name:  "URN",
			input: "urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
			valid: true,
		},
		{
			name:  "nil UUID",
			input: "00000000-0000-0000-0000-000000000000",
			valid: true,
		},
		{
			name:  "maximum UUID",
			input: "ffffffff-ffff-ffff-ffff-ffffffffffff",
			valid: true,
		},
		{
			name:  "version four",
			input: "550e8400-e29b-41d4-a716-446655440000",
			valid: true,
		},
		{
			name:  "version seven",
			input: "019a0abc-1234-7000-8000-000000000000",
			valid: true,
		},
		{
			name:  "version and variant unrestricted",
			input: "12345678-1234-f234-0234-123456789abc",
			valid: true,
		},
		{
			name: "empty",
		},
		{
			name:  "partial UUID",
			input: "f81d4fae",
		},
		{
			name:  "short compact hex",
			input: "f81d4fae7dec11d0a76500a0c91e6bf",
		},
		{
			name:  "long compact hex",
			input: "f81d4fae7dec11d0a76500a0c91e6bf60",
		},
		{
			name:  "short canonical",
			input: "f81d4fae-7dec-11d0-a765-00a0c91e6bf",
		},
		{
			name:  "long canonical",
			input: "f81d4fae-7dec-11d0-a765-00a0c91e6bf60",
		},
		{
			name:  "misplaced separator",
			input: "f81d4fa-e7dec-11d0-a765-00a0c91e6bf6",
		},
		{
			name:  "wrong separator",
			input: "f81d4fae_7dec-11d0-a765-00a0c91e6bf6",
		},
		{
			name:  "invalid hex",
			input: "g81d4fae-7dec-11d0-a765-00a0c91e6bf6",
		},
		{
			name:  "invalid compact hex",
			input: "f81d4fae7dec11d0a76500a0c91e6bfg",
		},
		{
			name:  "wrong closing brace",
			input: "{f81d4fae-7dec-11d0-a765-00a0c91e6bf6]",
		},
		{
			name:  "wrong opening brace",
			input: "[f81d4fae-7dec-11d0-a765-00a0c91e6bf6}",
		},
		{
			name:  "missing closing brace",
			input: "{f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
		},
		{
			name:  "uppercase URN prefix",
			input: "URN:UUID:f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
		},
		{
			name:  "wrong URN prefix",
			input: "urn:guid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
		},
		{
			name:  "braced compact hex",
			input: "{f81d4fae7dec11d0a76500a0c91e6bf6}",
		},
		{
			name:  "leading whitespace",
			input: " f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
		},
		{
			name:  "trailing newline",
			input: "f81d4fae-7dec-11d0-a765-00a0c91e6bf6\n",
		},
		{
			name:  "Unicode",
			input: "привет",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewUUID(nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			suite.Equal("uuid()", value.String())
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if test.valid {
				suite.Require().NoError(err)
				suite.Equal([]cobra.Completion{test.input}, candidates)
			} else {
				suite.Require().EqualError(err, "invalid uuid")
				suite.Empty(candidates)
			}
		})
	}
}

func (suite *UUIDTestSuite) TestConfiguration() {
	for _, test := range []struct {
		name          string
		properties    map[string]any
		subvalidators map[string][]any
		want          error
	}{
		{
			name: "nil maps",
		},
		{
			name:          "empty maps",
			properties:    map[string]any{},
			subvalidators: map[string][]any{},
		},
		{
			name: "unsupported property",
			properties: map[string]any{
				"format": "urn",
			},
			want: values.ErrNoProperties,
		},
		{
			name: "null property still unsupported",
			properties: map[string]any{
				"unknown": nil,
			},
			want: values.ErrNoProperties,
		},
		{
			name: "unsupported subvalidator",
			subvalidators: map[string][]any{
				"version": {int64(4)},
			},
			want: values.ErrNoSubvalidators,
		},
		{
			name: "empty subvalidator still unsupported",
			subvalidators: map[string][]any{
				"unknown": {},
			},
			want: values.ErrNoSubvalidators,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			properties := test.properties
			subvalidators := test.subvalidators

			// Act
			value, err := values.NewUUID(properties, subvalidators)
			factoryValue, factoryError := values.New("uuid", properties, subvalidators)

			// Assert
			if test.want != nil {
				suite.Require().ErrorIs(err, test.want)
				suite.Nil(value)

				suite.Require().ErrorIs(factoryError, test.want)
				suite.Nil(factoryValue)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(value)
				suite.Equal("uuid()", value.String())

				suite.Require().NoError(factoryError)
				suite.Require().NotNil(factoryValue)
				suite.Equal("uuid()", factoryValue.String())
			}
		})
	}
}

func TestUUID(t *testing.T) {
	t.Parallel()

	suite.Run(t, &UUIDTestSuite{})
}
