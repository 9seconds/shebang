package v1

import (
	"errors"
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type ArgsTestSuite struct {
	suite.Suite
}

func (suite *ArgsTestSuite) TestValidate() {
	for _, test := range []struct {
		name    string
		doc     string
		args    []string
		message string
	}{
		{
			name: "no arguments",
		},
		{
			name:    "unexpected arguments",
			args:    []string{"привет"},
			message: "expected 0 variadic arguments, got 1",
		},
		{
			name:    "missing leading argument",
			doc:     "arg \"source\"\n",
			message: "there must be at least 1 arguments, got 0",
		},
		{
			name: "fixed arguments",
			doc:  "arg \"source\"\narg \"dest\"\n",
			args: []string{"source", "dest"},
		},
		{
			name:    "too many fixed arguments",
			doc:     "arg \"source\"\n",
			args:    []string{"source", "extra"},
			message: "expected 0 variadic arguments, got 1",
		},
		{
			name: "empty unlimited vararg",
			doc:  "vararg \"items\"\n",
		},
		{
			name: "unlimited vararg",
			doc:  "vararg \"items\"\n",
			args: []string{"one", "two", "three"},
		},
		{
			name:    "minimum not met",
			doc:     "arg \"source\"\nvararg \"items\" { min-count 2; }\narg \"dest\"\n",
			args:    []string{"source", "one", "dest"},
			message: "there must be at least 4 arguments, got 3",
		},
		{
			name: "minimum met",
			doc:  "vararg \"items\" { min-count 2; }\n",
			args: []string{"one", "two"},
		},
		{
			name: "maximum met with required items",
			doc:  "arg \"source\"\nvararg \"items\" {\nmin-count 1\nmax-count 2\n}\narg \"dest\"\n",
			args: []string{"source", "one", "two", "dest"},
		},
		{
			name:    "maximum includes required items",
			doc:     "vararg \"items\" {\nmin-count 1\nmax-count 2\n}\n",
			args:    []string{"one", "two", "three"},
			message: "there must be at most 2 variadic arguments, got 3",
		},
		{
			name: "zero maximum accepts empty group",
			doc:  "vararg \"items\" { max-count 0; }\n",
		},
		{
			name:    "zero maximum rejects item",
			doc:     "vararg \"items\" { max-count 0; }\n",
			args:    []string{"one"},
			message: "there must be at most 0 variadic arguments, got 1",
		},
		{
			name: "negative counts impose no limits",
			doc:  "vararg \"items\" {\nmin-count -2\nmax-count -1\n}\n",
			args: []string{"one", "two", "three"},
		},
		{
			name:    "missing trailing argument",
			doc:     "vararg \"items\"\narg \"dest\"\n",
			message: "there must be at least 1 arguments, got 0",
		},
		{
			name: "trailing arguments with empty vararg",
			doc:  "vararg \"items\"\narg \"dest\"\n",
			args: []string{"dest"},
		},
		{
			name: "groups use their own validators",
			doc: `arg "source" {
				value "str" { re "^source$"; }
			}
			vararg "items" {
				min-count 1
				value "str" { re "^привет$"; }
			}
			arg "dest" {
				value "str" { re "^dest$"; }
			}
			arg "end" {
				value "str" { re "^end$"; }
			}
			`,
			args: []string{"source", "привет", "привет", "dest", "end"},
		},
		{
			name:    "leading validation error",
			doc:     "arg \"word\" { value \"str\" { re \"^ok$\"; }; }\n",
			args:    []string{"bad"},
			message: "invalid argument WORD: bad does not match ^ok$",
		},
		{
			name:    "trailing validation error",
			doc:     "vararg \"items\"\narg \"dest\" { value \"str\" { re \"^ok$\"; }; }\n",
			args:    []string{"one", "bad"},
			message: "invalid argument DEST: bad does not match ^ok$",
		},
		{
			name:    "required vararg validation error",
			doc:     "vararg \"items\" {\nmin-count 2\nvalue \"str\" { re \"^ok$\"; }\n}\n",
			args:    []string{"ok", "bad"},
			message: "invalid argument ITEMS2: bad does not match ^ok$",
		},
		{
			name:    "optional vararg validation error",
			doc:     "vararg \"items\" { value \"str\" { re \"^ok$\"; }; }\n",
			args:    []string{"ok", "bad"},
			message: "invalid argument ITEMS2: bad does not match ^ok$",
		},
		{
			name:    "optional vararg numbering includes minimum",
			doc:     "vararg \"items\" {\nmin-count 2\nvalue \"str\" { re \"^ok$\"; }\n}\n",
			args:    []string{"ok", "ok", "ok", "bad"},
			message: "invalid argument ITEMS4: bad does not match ^ok$",
		},
		{
			name: "unicode length counts runes",
			doc:  "arg \"word\" { value \"str\" { max-length 6; }; }\n",
			args: []string{"привет"},
		},
	} {
		suite.Run(test.name, func() {
			conf, err := Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)

			validators, err := newArgValidators(conf)
			suite.Require().NoError(err)

			err = validators.Validate(test.args)

			if test.message == "" {
				suite.NoError(err)
			} else {
				suite.EqualError(err, test.message)
			}
		})
	}
}

func (suite *ArgsTestSuite) TestValidationErrorWrapping() {
	want := errors.New("invalid value")

	for _, test := range []struct {
		name       string
		validators argValidators
		args       []string
		message    string
	}{
		{
			name: "leading",
			validators: argValidators{
				first: []argValidator{
					{
						name: "FIRST",
						validator: &argsTestValue{
							err: want,
						},
					},
				},
			},
			args:    []string{"value"},
			message: "invalid argument FIRST:",
		},
		{
			name: "trailing",
			validators: argValidators{
				last: []argValidator{
					{
						name: "LAST",
						validator: &argsTestValue{
							err: want,
						},
					},
				},
			},
			args:    []string{"value"},
			message: "invalid argument LAST:",
		},
		{
			name: "variadic",
			validators: argValidators{
				varArg: &varArgValidator{
					argValidator: argValidator{
						name: "ITEMS",
						validator: &argsTestValue{
							err: want,
						},
					},
					maxCount: -1,
				},
			},
			args:    []string{"value"},
			message: "invalid argument ITEMS1:",
		},
	} {
		suite.Run(test.name, func() {
			err := test.validators.Validate(test.args)
			suite.ErrorIs(err, want)
			suite.ErrorContains(err, test.message)
		})
	}
}

func (suite *ArgsTestSuite) TestComplete() {
	for _, test := range []struct {
		name     string
		doc      string
		args     []string
		selected string
	}{
		{
			name: "no positional arguments",
		},
		{
			name:     "first leading argument",
			doc:      "arg \"first\"\narg \"second\"\n",
			selected: "FIRST",
		},
		{
			name:     "next leading argument",
			doc:      "arg \"first\"\narg \"second\"\n",
			args:     []string{"one"},
			selected: "SECOND",
		},
		{
			name: "fixed capacity reached",
			doc:  "arg \"first\"\n",
			args: []string{"one"},
		},
		{
			name: "fixed capacity exceeded",
			doc:  "arg \"first\"\n",
			args: []string{"one", "two"},
		},
		{
			name:     "required vararg item",
			doc:      "arg \"first\"\nvararg \"items\" { min-count 2; }\n",
			args:     []string{"one", "two"},
			selected: "ITEMS2",
		},
		{
			name:     "unlimited vararg",
			doc:      "vararg \"items\"\n",
			args:     []string{"one", "two", "three"},
			selected: "ITEMS",
		},
		{
			name:     "optional bounded vararg",
			doc:      "vararg \"items\" {\nmin-count 1\nmax-count 3\n}\n",
			args:     []string{"one", "two"},
			selected: "ITEMS",
		},
		{
			name: "bounded vararg capacity reached",
			doc:  "vararg \"items\" {\nmin-count 1\nmax-count 3\n}\n",
			args: []string{"one", "two", "three"},
		},
		{
			name: "zero vararg capacity",
			doc:  "vararg \"items\" { max-count 0; }\n",
		},
		{
			name:     "negative maximum is unlimited after required items",
			doc:      "vararg \"items\" {\nmin-count 1\nmax-count -1\n}\n",
			args:     []string{"one", "two", "three"},
			selected: "ITEMS",
		},
		{
			name:     "negative minimum with finite capacity",
			doc:      "vararg \"items\" {\nmin-count -1\nmax-count 2\n}\n",
			args:     []string{"one"},
			selected: "ITEMS",
		},
		{
			name: "normalized finite capacity reached",
			doc:  "vararg \"items\" {\nmin-count -1\nmax-count 2\n}\n",
			args: []string{"one", "two"},
		},
		{
			name:     "first trailing argument preferred",
			doc:      "arg \"first\"\nvararg \"items\"\narg \"dest\"\narg \"end\"\n",
			args:     []string{"one"},
			selected: "DEST",
		},
		{
			name:     "next trailing argument",
			doc:      "arg \"first\"\nvararg \"items\"\narg \"dest\"\narg \"end\"\n",
			args:     []string{"one", "two"},
			selected: "END",
		},
		{
			name:     "last trailing validator reused",
			doc:      "vararg \"items\"\narg \"dest\"\narg \"end\"\n",
			args:     []string{"one", "two", "three", "four"},
			selected: "END",
		},
		{
			name:     "trailing after required varargs",
			doc:      "vararg \"items\" { min-count 1; }\narg \"dest\"\n",
			args:     []string{"one"},
			selected: "DEST",
		},
		{
			name:     "bounded total capacity includes trailing arguments",
			doc:      "arg \"first\"\nvararg \"items\" {\nmin-count 1\nmax-count 2\n}\narg \"dest\"\n",
			args:     []string{"one", "two", "three"},
			selected: "DEST",
		},
		{
			name: "bounded total capacity reached",
			doc:  "arg \"first\"\nvararg \"items\" {\nmin-count 1\nmax-count 2\n}\narg \"dest\"\n",
			args: []string{"one", "two", "three", "four"},
		},
	} {
		suite.Run(test.name, func() {
			conf, err := Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)

			validators, err := newArgValidators(conf)
			suite.Require().NoError(err)

			calls := []string(nil)
			for idx := range validators.first {
				validators.first[idx].validator = &argsTestValue{
					name:  validators.first[idx].name,
					calls: &calls,
				}
			}

			for idx := range validators.last {
				validators.last[idx].validator = &argsTestValue{
					name:  validators.last[idx].name,
					calls: &calls,
				}
			}

			if validators.varArg != nil {
				validators.varArg.validator = &argsTestValue{
					name:  validators.varArg.name,
					calls: &calls,
				}
			}

			completions, directive := validators.Complete(test.args, "привет")

			if test.selected == "" {
				suite.Nil(completions)
				suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
				suite.Empty(calls)
			} else {
				suite.Equal([]cobra.Completion{test.selected + ":привет"}, completions)
				suite.Equal(cobra.ShellCompDirectiveNoSpace, directive)
				suite.Equal([]string{test.selected}, calls)
			}
		})
	}
}

func (suite *ArgsTestSuite) TestNewArgValidators() {
	for _, test := range []struct {
		name   string
		doc    string
		first  []string
		last   []string
		vararg string
		min    int
		max    int
	}{
		{
			name: "empty configuration",
		},
		{
			name:  "fixed argument order",
			doc:   "arg \"second\"\narg \"first\"\n",
			first: []string{"SECOND", "FIRST"},
		},
		{
			name:   "unbounded vararg defaults",
			doc:    "vararg \"items\"\n",
			vararg: "ITEMS",
			max:    -1,
		},
		{
			name:   "required items follow leading arguments",
			doc:    "arg \"source\"\nvararg \"items\" {\nmin-count 2\nmax-count 4\n}\narg \"dest\"\narg \"end\"\n",
			first:  []string{"SOURCE", "ITEMS1", "ITEMS2"},
			last:   []string{"DEST", "END"},
			vararg: "ITEMS",
			min:    2,
			max:    4,
		},
		{
			name:   "zero minimum and maximum",
			doc:    "vararg \"items\" {\nmin-count 0\nmax-count 0\n}\n",
			vararg: "ITEMS",
		},
		{
			name:   "negative minimum normalized",
			doc:    "vararg \"items\" { min-count -2; }\n",
			vararg: "ITEMS",
			max:    -1,
		},
	} {
		suite.Run(test.name, func() {
			conf, err := Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)

			validators, err := newArgValidators(conf)
			suite.Require().NoError(err)
			suite.Require().NotNil(validators)

			first := []string(nil)

			for _, arg := range validators.first {
				first = append(first, arg.name)
				suite.NotNil(arg.validator)
			}

			last := []string(nil)
			for _, arg := range validators.last {
				last = append(last, arg.name)
				suite.NotNil(arg.validator)
			}

			suite.Equal(test.first, first)
			suite.Equal(test.last, last)

			if test.vararg == "" {
				suite.Nil(validators.varArg)
			} else {
				suite.Require().NotNil(validators.varArg)
				suite.Equal(test.vararg, validators.varArg.name)
				suite.Equal(test.min, validators.varArg.minCount)
				suite.Equal(test.max, validators.varArg.maxCount)
				suite.NotNil(validators.varArg.validator)
			}
		})
	}
}

func (suite *ArgsTestSuite) TestNewArgValidatorsErrors() {
	for _, test := range []struct {
		name    string
		doc     string
		message string
		want    error
	}{
		{
			name:    "leading argument type",
			doc:     "arg \"source\" { value \"unknown\"; }\n",
			message: "invalid argument SOURCE:",
			want:    values.ErrUnknownValueType,
		},
		{
			name:    "required vararg type",
			doc:     "vararg \"items\" {\nmin-count 1\nvalue \"unknown\"\n}\n",
			message: "invalid vararg type:",
			want:    values.ErrUnknownValueType,
		},
		{
			name:    "optional vararg type",
			doc:     "vararg \"items\" { value \"unknown\"; }\n",
			message: "invalid argument ITEMS:",
			want:    values.ErrUnknownValueType,
		},
		{
			name:    "trailing argument type",
			doc:     "vararg \"items\"\narg \"dest\" { value \"unknown\"; }\n",
			message: "invalid argument DEST:",
			want:    values.ErrUnknownValueType,
		},
		{
			name:    "invalid validator property",
			doc:     "arg \"source\" { value \"str\" { unknown 1; }; }\n",
			message: "invalid argument SOURCE:",
			want:    values.ErrUnknownProperty,
		},
	} {
		suite.Run(test.name, func() {
			conf, err := Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)

			validators, err := newArgValidators(conf)
			suite.Require().Error(err)
			suite.ErrorContains(err, test.message)
			suite.ErrorIs(err, test.want)
			suite.Nil(validators)
		})
	}
}

type argsTestValue struct {
	name  string
	err   error
	calls *[]string
}

func (v *argsTestValue) Validate(string) error {
	return v.err
}

func (v *argsTestValue) Complete(prefix string) ([]cobra.Completion, cobra.ShellCompDirective) {
	*v.calls = append(*v.calls, v.name)
	return []cobra.Completion{v.name + ":" + prefix}, cobra.ShellCompDirectiveNoSpace
}

func (v *argsTestValue) String() string {
	return v.name
}

func TestArgs(t *testing.T) {
	suite.Run(t, &ArgsTestSuite{})
}
