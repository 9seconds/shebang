package values_test

import (
	"regexp/syntax"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type EmailTestSuite struct {
	suite.Suite
}

func (suite *EmailTestSuite) TestParseAndComplete() {
	for _, test := range []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "ordinary address",
			input: "alice@example.com",
			valid: true,
		},
		{
			name:  "case preserved",
			input: "Alice@EXAMPLE.COM",
			valid: true,
		},
		{
			name:  "plus addressing and subdomain",
			input: "first.last+tag@mail.example.com",
			valid: true,
		},
		{
			name:  "quoted local part",
			input: `"alice"@example.com`,
			valid: true,
		},
		{
			name:  "IPv4 literal",
			input: "alice@[192.0.2.1]",
			valid: true,
		},
		{
			name:  "unknown suffix allowed by default",
			input: "alice@example.invalid",
			valid: true,
		},
		{
			name:  "punycode domain",
			input: "alice@xn--bcher-kva.de",
			valid: true,
		},
		{
			name: "empty",
		},
		{
			name:  "missing at sign",
			input: "alice.example.com",
		},
		{
			name:  "missing local part",
			input: "@example.com",
		},
		{
			name:  "missing domain",
			input: "alice@",
		},
		{
			name:  "single label domain",
			input: "alice@localhost",
		},
		{
			name:  "consecutive local dots",
			input: "first..last@example.com",
		},
		{
			name:  "leading local dot",
			input: ".alice@example.com",
		},
		{
			name:  "trailing local dot",
			input: "alice.@example.com",
		},
		{
			name:  "domain underscore",
			input: "alice@bad_domain.com",
		},
		{
			name:  "domain leading hyphen",
			input: "alice@-example.com",
		},
		{
			name:  "display name",
			input: "Alice <alice@example.com>",
		},
		{
			name:  "leading whitespace",
			input: " alice@example.com",
		},
		{
			name:  "trailing newline",
			input: "alice@example.com\n",
		},
		{
			name:  "Unicode local part",
			input: "привет@example.com",
		},
		{
			name:  "Unicode domain",
			input: "alice@пример.рф",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewEmail(nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			suite.Equal("email()", value.String())
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if test.valid {
				suite.Require().NoError(err)
				suite.Equal([]cobra.Completion{test.input}, candidates)
			} else {
				suite.Require().Error(err)
				suite.Empty(candidates)

				if test.input == "" {
					suite.Require().ErrorIs(err, values.ErrEmailEmpty)
				}
			}
		})
	}
}

func (suite *EmailTestSuite) TestConstraints() {
	for _, test := range []struct {
		name          string
		subvalidators map[string][]any
		input         string
		want          error
	}{
		{
			name: "exact domain",
			subvalidators: map[string][]any{
				"domain": {"example.com"},
			},
			input: "alice@example.com",
		},
		{
			name: "domain case sensitive",
			subvalidators: map[string][]any{
				"domain": {"example.com"},
			},
			input: "alice@EXAMPLE.COM",
			want:  values.ErrEmailDomain,
		},
		{
			name: "subdomain is not exact domain",
			subvalidators: map[string][]any{
				"domain": {"example.com"},
			},
			input: "alice@mail.example.com",
			want:  values.ErrEmailDomain,
		},
		{
			name: "empty exact domain accepts no address",
			subvalidators: map[string][]any{
				"domain": {""},
			},
			input: "alice@example.com",
			want:  values.ErrEmailDomain,
		},
		{
			name: "domain regex match",
			subvalidators: map[string][]any{
				"domain-re": {`(^|\.)example\.com$`},
			},
			input: "alice@mail.example.com",
		},
		{
			name: "domain regex rejection",
			subvalidators: map[string][]any{
				"domain-re": {`^example\.com$`},
			},
			input: "alice@example.org",
			want:  values.ErrEmailDomain,
		},
		{
			name: "unanchored domain regex",
			subvalidators: map[string][]any{
				"domain-re": {"example"},
			},
			input: "alice@notexample.com",
		},
		{
			name: "user regex match",
			subvalidators: map[string][]any{
				"user-re": {`^alice(\+.*)?$`},
			},
			input: "alice+tag@example.com",
		},
		{
			name: "user regex case sensitive",
			subvalidators: map[string][]any{
				"user-re": {"^alice$"},
			},
			input: "Alice@example.com",
			want:  values.ErrEmailUser,
		},
		{
			name: "quoted local part includes quotes",
			subvalidators: map[string][]any{
				"user-re": {`^"alice"$`},
			},
			input: `"alice"@example.com`,
		},
		{
			name: "user regex does not see domain",
			subvalidators: map[string][]any{
				"user-re": {"example"},
			},
			input: "alice@example.com",
			want:  values.ErrEmailUser,
		},
		{
			name: "empty regexes impose no restriction",
			subvalidators: map[string][]any{
				"domain-re": {""},
				"user-re":   {""},
			},
			input: "alice@example.com",
		},
		{
			name: "ICANN required matches uppercase suffix",
			subvalidators: map[string][]any{
				"icann": {true},
			},
			input: "alice@EXAMPLE.COM",
		},
		{
			name: "ICANN required rejects unknown suffix",
			subvalidators: map[string][]any{
				"icann": {true},
			},
			input: "alice@example.invalid",
			want:  values.ErrEmailICANNRequired,
		},
		{
			name: "ICANN excluded permits unknown suffix",
			subvalidators: map[string][]any{
				"icann": {false},
			},
			input: "alice@example.invalid",
		},
		{
			name: "ICANN excluded rejects managed suffix",
			subvalidators: map[string][]any{
				"icann": {false},
			},
			input: "alice@example.com",
			want:  values.ErrEmailICANNExcluded,
		},
		{
			name: "private public suffix is not ICANN",
			subvalidators: map[string][]any{
				"icann": {false},
			},
			input: "alice@project.blogspot.com",
		},
		{
			name: "all constraints combined",
			subvalidators: map[string][]any{
				"domain":    {"example.com"},
				"domain-re": {`^example\.com$`},
				"user-re":   {"^alice$"},
				"icann":     {true},
			},
			input: "alice@example.com",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.New("email", nil, test.subvalidators)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if test.want == nil {
				suite.Require().NoError(err)
				suite.Equal([]cobra.Completion{test.input}, candidates)
			} else {
				suite.Require().ErrorIs(err, test.want)
				suite.NotEmpty(err.Error())
				suite.Empty(candidates)
			}
		})
	}
}

func (suite *EmailTestSuite) TestSubvalidatorArguments() {
	for _, name := range []string{
		"domain",
		"domain-re",
		"user-re",
		"icann",
	} {
		suite.Run(name, func() {
			argument := any("example")
			expectedType := "string"

			if name == "icann" {
				argument = true
				expectedType = "bool"
			}

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
					args: []any{argument, argument},
					want: utils.ErrSingleArgumentExpected,
				},
				{
					name: "integer argument",
					args: []any{int64(1)},
				},
				{
					name: "null argument",
					args: []any{nil},
				},
			} {
				suite.Run(test.name, func() {
					// Arrange
					subvalidators := map[string][]any{
						name: test.args,
					}

					// Act
					value, err := values.NewEmail(nil, subvalidators)

					// Assert
					suite.Nil(value)
					suite.Require().ErrorContains(err, "cannot add subvalidator "+name)

					if test.want != nil {
						suite.Require().ErrorIs(err, test.want)
					} else {
						var typeError *utils.ArgumentTypeError

						suite.Require().ErrorAs(err, &typeError)
						suite.Equal(expectedType, typeError.Expected)
					}
				})
			}
		})
	}
}

func (suite *EmailTestSuite) TestInvalidRegex() {
	for _, name := range []string{
		"domain-re",
		"user-re",
	} {
		suite.Run(name, func() {
			// Arrange
			subvalidators := map[string][]any{
				name: {"["},
			}

			// Act
			value, err := values.NewEmail(nil, subvalidators)

			// Assert
			var regexError *syntax.Error

			suite.Nil(value)
			suite.Require().ErrorAs(err, &regexError)
			suite.Equal("[", regexError.Expr)
		})
	}
}

func (suite *EmailTestSuite) TestUnknownSubvalidator() {
	// Arrange
	subvalidators := map[string][]any{
		"unknown": {},
	}

	// Act
	value, err := values.New("email", nil, subvalidators)

	// Assert
	suite.Nil(value)
	suite.Require().ErrorIs(err, values.ErrUnknownSubvalidator)
}

func (suite *EmailTestSuite) TestString() {
	// Arrange
	value, err := values.NewEmail(nil, map[string][]any{
		"user-re":   {"^alice$"},
		"domain-re": {`^example\.com$`},
		"domain":    {"example.com"},
		"icann":     {false},
	})
	suite.Require().NoError(err)

	// Act
	description := value.String()

	// Assert
	suite.Equal(`email(domain-re:^example\.com$, domain:example.com, icann:false, user-re:^alice$)`,
		description)
}

func TestEmail(t *testing.T) {
	t.Parallel()

	suite.Run(t, &EmailTestSuite{})
}
