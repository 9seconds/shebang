package values_test

import (
	"net/url"
	"regexp/syntax"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type URLTestSuite struct {
	suite.Suite
}

func (suite *URLTestSuite) TestParsing() {
	for _, test := range []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "empty reference",
			valid: true,
		},
		{
			name:  "relative path",
			input: "../привет",
			valid: true,
		},
		{
			name:  "host-looking path",
			input: "example.com/path",
			valid: true,
		},
		{
			name:  "query reference",
			input: "?x=1",
			valid: true,
		},
		{
			name:  "opaque URL",
			input: "mailto:alice@example.com",
			valid: true,
		},
		{
			name:  "absolute URL preserves spelling",
			input: "HTTPS://Alice:p%40ss@EXAMPLE.COM:443/a%2Fb?q=a+b#%D0%BF",
			valid: true,
		},
		{
			name:  "IPv6 authority",
			input: "//[2001:db8::1]:8080/path",
			valid: true,
		},
		{
			name:  "query escapes deferred until query validation",
			input: "https://example.com/?x=%zz",
			valid: true,
		},
		{
			name:  "invalid path escape",
			input: "/%zz",
		},
		{
			name:  "invalid fragment escape",
			input: "/#%zz",
		},
		{
			name:  "invalid port",
			input: "https://example.com:bad/",
		},
		{
			name:  "control character",
			input: "https://example.com/\n",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewURL(nil, nil)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)
			candidates, directive := value.Complete(test.input)

			// Assert
			suite.Equal("url()", value.String())
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			if test.valid {
				suite.Require().NoError(err)
				suite.Equal([]cobra.Completion{test.input}, candidates)
			} else {
				var parseError *url.Error

				suite.Require().ErrorAs(err, &parseError)
				suite.Equal(test.input, parseError.URL)
				suite.Empty(candidates)
			}
		})
	}
}

func (suite *URLTestSuite) TestComponentConstraints() {
	for _, test := range []struct {
		name     string
		args     []any
		accepted []string
		rejected []string
	}{
		{
			name: "scheme",
			args: []any{"https"},
			accepted: []string{
				"HTTPS://example.com",
				"https:opaque",
			},
			rejected: []string{
				"http://example.com",
				"/relative",
			},
		},
		{
			name:     "user",
			args:     []any{"alice"},
			accepted: []string{"https://%61lice@example.com"},
			rejected: []string{
				"https://bob@example.com",
				"https://example.com",
			},
		},
		{
			name:     "user-re",
			args:     []any{"^alice$"},
			accepted: []string{"https://alice@example.com"},
			rejected: []string{
				"https://Alice@example.com",
				"https://example.com",
			},
		},
		{
			name: "password",
			args: []any{true},
			accepted: []string{
				"https://alice:secret@example.com",
				"https://alice:@example.com",
			},
			rejected: []string{
				"https://alice@example.com",
				"https://example.com",
			},
		},
		{
			name: "password",
			args: []any{false},
			accepted: []string{
				"https://alice@example.com",
				"https://example.com",
			},
			rejected: []string{"https://alice:@example.com"},
		},
		{
			name:     "domain",
			args:     []any{"example.com"},
			accepted: []string{"https://example.com:8443"},
			rejected: []string{
				"https://EXAMPLE.COM",
				"https://sub.example.com",
				"/relative",
			},
		},
		{
			name:     "domain",
			args:     []any{"2001:db8::1"},
			accepted: []string{"https://[2001:db8::1]:8080"},
			rejected: []string{"https://[2001:db8::2]"},
		},
		{
			name:     "domain-re",
			args:     []any{`(?i)(^|\.)example\.com$`},
			accepted: []string{"https://SUB.EXAMPLE.COM:8443"},
			rejected: []string{"https://example.com.evil.test"},
		},
		{
			name:     "path-re",
			args:     []any{"^/a/b$"},
			accepted: []string{"https://example.com/a%2Fb"},
			rejected: []string{"mailto:alice@example.com"},
		},
		{
			name: "path-glob",
			args: []any{"/api/*.json"},
			accepted: []string{
				"/api/report.json",
				"/api/привет.json",
			},
			rejected: []string{
				"/api/nested/report.json",
				"/api/report.txt",
			},
		},
		{
			name:     "path-glob",
			args:     []any{"/api/**/*.json"},
			accepted: []string{"/api/nested/report.json"},
			rejected: []string{"/other/report.json"},
		},
		{
			name:     "fragment-re",
			args:     []any{"^привет$"},
			accepted: []string{"/#%D0%BF%D1%80%D0%B8%D0%B2%D0%B5%D1%82"},
			rejected: []string{"/#other"},
		},
	} {
		suite.Run(test.name, func() {
			for _, valid := range []bool{true, false} {
				inputs := test.rejected
				if valid {
					inputs = test.accepted
				}

				for _, input := range inputs {
					suite.Run(input, func() {
						// Arrange
						value, err := values.New("url", nil, map[string][]any{
							test.name: test.args,
						})
						suite.Require().NoError(err)

						// Act
						err = value.Validate(input)
						candidates, directive := value.Complete(input)

						// Assert
						suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

						if valid {
							suite.Require().NoError(err)
							suite.Equal([]cobra.Completion{input}, candidates)

							return
						}

						suite.Require().Error(err)
						suite.Empty(candidates)

						if test.name == "password" {
							want := values.ErrURLPasswordExcluded
							if test.args[0] == true {
								want = values.ErrURLPasswordRequired
							}

							suite.Require().ErrorIs(err, want)
						} else {
							var constraintError *values.ErrURLConstraint

							suite.Require().ErrorAs(err, &constraintError)
						}
					})
				}
			}
		})
	}
}

func (suite *URLTestSuite) TestQueries() {
	for _, name := range []string{
		"query",
		"query-re",
	} {
		suite.Run(name, func() {
			for _, test := range []struct {
				input string
				valid bool
				want  error
			}{
				{
					input: "?q=a+b&q=a%20b&empty&extra=ignored",
					valid: true,
				},
				{
					input: "?q=a+b&q=wrong&empty=",
				},
				{
					input: "?q=wrong&q=a+b&empty=",
				},
				{
					input: "?empty=",
					want:  values.ErrURLQueryMissingKey,
				},
				{
					input: "?q=a+b&empty=&extra=%zz",
				},
				{
					input: "?q=a+b&empty=&extra=a;b",
				},
			} {
				suite.Run(test.input, func() {
					// Arrange
					pattern := "a b"
					if name == "query-re" {
						pattern = "^a b$"
					}

					value, err := values.NewURL(nil, map[string][]any{
						name: {
							"q",
							pattern,
							"empty",
							"",
						},
					})
					suite.Require().NoError(err)

					// Act
					err = value.Validate(test.input)
					candidates, directive := value.Complete(test.input)

					// Assert
					suite.Equal("url("+name+":q="+pattern+",empty=)", value.String())
					suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

					if test.valid {
						suite.Require().NoError(err)
						suite.Equal([]cobra.Completion{test.input}, candidates)
					} else {
						suite.Require().Error(err)
						suite.Empty(candidates)

						if test.want != nil {
							suite.Require().ErrorIs(err, test.want)
						}
					}
				})
			}
		})
	}
}

func (suite *URLTestSuite) TestConfigurationErrors() {
	for _, name := range []string{
		"scheme",
		"user",
		"user-re",
		"password",
		"domain",
		"domain-re",
		"path-re",
		"path-glob",
		"query",
		"query-re",
		"fragment-re",
	} {
		suite.Run(name, func() {
			for _, args := range [][]any{
				nil,
				{int64(1)},
				{nil},
			} {
				// Arrange
				subvalidators := map[string][]any{
					name: args,
				}

				// Act
				value, err := values.NewURL(nil, subvalidators)

				// Assert
				suite.Nil(value)
				suite.Require().ErrorContains(err, "incorrect subvalidator "+name)

				if len(args) == 0 {
					want := utils.ErrEmpty
					if name == "query" || name == "query-re" {
						want = values.ErrURLQueryEmpty
					}

					suite.Require().ErrorIs(err, want)
				} else {
					var typeError *utils.ArgumentTypeError

					suite.Require().ErrorAs(err, &typeError)
				}
			}
		})
	}
}

func (suite *URLTestSuite) TestMultipleArguments() {
	for _, name := range []string{
		"scheme",
		"user",
		"user-re",
		"password",
		"domain",
		"domain-re",
		"path-re",
		"path-glob",
		"fragment-re",
	} {
		suite.Run(name, func() {
			// Arrange
			args := []any{
				"one",
				"two",
			}
			if name == "password" {
				args = []any{true, false}
			}

			subvalidators := map[string][]any{
				name: args,
			}

			// Act
			value, err := values.NewURL(nil, subvalidators)

			// Assert
			suite.Nil(value)
			suite.Require().ErrorIs(err, utils.ErrSingleArgumentExpected)
		})
	}
}

func (suite *URLTestSuite) TestInvalidPatterns() {
	for _, name := range []string{
		"user-re",
		"domain-re",
		"path-re",
		"fragment-re",
		"query-re",
		"path-glob",
	} {
		suite.Run(name, func() {
			// Arrange
			args := []any{"["}
			if name == "query-re" {
				args = []any{
					"key",
					"[",
				}
			}

			// Act
			value, err := values.NewURL(nil, map[string][]any{
				name: args,
			})

			// Assert
			suite.Nil(value)

			if name == "path-glob" {
				suite.Require().ErrorIs(err, values.ErrURLInvalidGlob)
			} else {
				var regexError *syntax.Error

				suite.Require().ErrorAs(err, &regexError)
			}
		})
	}
}

func (suite *URLTestSuite) TestQueryConfiguration() {
	for _, name := range []string{
		"query",
		"query-re",
	} {
		for _, test := range []struct {
			args []any
			want error
		}{
			{
				args: []any{"key"},
				want: values.ErrURLQueryMissingValue,
			},
			{
				args: []any{
					"key",
					"a",
					"key",
					"b",
				},
				want: values.ErrURLQueryDuplicateKey,
			},
		} {
			suite.Run(name+test.want.Error(), func() {
				// Arrange
				subvalidators := map[string][]any{
					name: test.args,
				}

				// Act
				value, err := values.NewURL(nil, subvalidators)

				// Assert
				suite.Nil(value)
				suite.Require().ErrorIs(err, test.want)
			})
		}
	}
}

func (suite *URLTestSuite) TestUnknownSubvalidator() {
	// Arrange
	subvalidators := map[string][]any{
		"unknown": {},
	}

	// Act
	value, err := values.New("url", nil, subvalidators)

	// Assert
	suite.Nil(value)
	suite.Require().ErrorIs(err, values.ErrUnknownSubvalidator)
}

func (suite *URLTestSuite) TestConstraintDetails() {
	// Arrange
	value, err := values.NewURL(nil, map[string][]any{
		"user": {"alice"},
	})
	suite.Require().NoError(err)

	// Act
	err = value.Validate("https://bob@example.com")

	// Assert
	var constraintError *values.ErrURLConstraint

	suite.Require().ErrorAs(err, &constraintError)
	suite.Equal("user", constraintError.Component)
	suite.Equal("equal", constraintError.Operation)
	suite.Equal("alice", constraintError.Expected)
	suite.Equal("bob", constraintError.Actual)
	suite.Equal(`URL user must equal "alice", got "bob"`, constraintError.Error())
}

func TestURL(t *testing.T) {
	t.Parallel()

	suite.Run(t, &URLTestSuite{})
}
