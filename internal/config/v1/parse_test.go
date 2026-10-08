package v1_test

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	v1 "github.com/9seconds/shebang/internal/config/v1"
	"github.com/9seconds/shebang/internal/utils"
	"github.com/stretchr/testify/suite"
)

type ParseTestSuite struct {
	suite.Suite
}

func (suite *ParseTestSuite) TestMetadata() {
	for _, test := range []struct {
		name        string
		doc         string
		description string
		example     string
		argv        []string
	}{
		{
			name: "empty input",
			argv: []string{"bash"},
		},
		{
			name: "comments only",
			doc:  "// configuration\n/* empty */\n",
			argv: []string{"bash"},
		},
		{
			name:        "description and example",
			doc:         "description \"A script\"\nexample \"script source dest\"\n",
			description: "A script",
			example:     "script source dest",
			argv:        []string{"bash"},
		},
		{
			name: "execute arguments",
			doc:  "execute \"bash\" \"-eu\" \"-o\" \"pipefail\"\n",
			argv: []string{"bash", "-eu", "-o", "pipefail"},
		},
		{
			name:        "last scalar wins",
			doc:         "description \"old\"\ndescription \"new\"\nexample \"old\"\nexample \"new\"\nexecute \"bash\" \"-x\"\nexecute \"sh\"\n",
			description: "new",
			example:     "new",
			argv:        []string{"sh"},
		},
		{
			name: "empty metadata",
			doc:  "description \"\"\nexample \"\"\n",
			argv: []string{"bash"},
		},
		{
			name:        "unicode metadata",
			doc:         "description \"привет\"\nexample \"привет\"\n",
			description: "привет",
			example:     "привет",
			argv:        []string{"bash"},
		},
		{
			name: "empty execute replaced",
			doc:  "execute\nexecute \"sh\"\n",
			argv: []string{"sh"},
		},
	} {
		suite.Run(test.name, func() {
			conf, err := v1.Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)
			suite.Require().NotNil(conf)
			suite.Equal(test.description, conf.Description)
			suite.Equal(test.example, conf.Example)
			suite.Equal(test.argv, conf.Argv)
			suite.Nil(conf.Options)
			suite.Nil(conf.Flags)
			suite.Nil(conf.FirstArgs)
			suite.Nil(conf.LastArgs)
			suite.Nil(conf.VarArgs)
		})
	}
}

func (suite *ParseTestSuite) TestOptionsAndFlags() {
	for _, test := range []struct {
		name    string
		doc     string
		options []v1.Option
		flags   []v1.Flag
	}{
		{
			name: "defaults",
			doc:  "option \"output\"\nflag \"verbose\"\n",
			options: []v1.Option{
				{
					Flag: v1.Flag{
						Name: "output",
					},
				},
			},
			flags: []v1.Flag{
				{
					Name: "verbose",
				},
			},
		},
		{
			name: "configured",
			doc: `option "output" {
				description "Output file"
				short "o"
				value "str" {
					min-length 1
				}
			}
			flag "verbose" {
				description "Verbose output"
				short "v"
			}
			`,
			options: []v1.Option{
				{
					Flag: v1.Flag{
						Name:        "output",
						Description: "Output file",
						Short:       "o",
					},
					WithValue: v1.WithValue{
						Type: "str",
						Properties: map[string][]any{
							"min-length": {int64(1)},
						},
					},
				},
			},
			flags: []v1.Flag{
				{
					Name:        "verbose",
					Description: "Verbose output",
					Short:       "v",
				},
			},
		},
		{
			name: "declaration order",
			doc:  "flag \"z\" { short \"z\"; }\noption \"b\" { short \"b\"; }\nflag \"c\"\noption \"a\"\n",
			options: []v1.Option{
				{
					Flag: v1.Flag{
						Name:  "b",
						Short: "b",
					},
				},
				{
					Flag: v1.Flag{
						Name: "a",
					},
				},
			},
			flags: []v1.Flag{
				{
					Name:  "z",
					Short: "z",
				},
				{
					Name: "c",
				},
			},
		},
		{
			name: "repeated fields",
			doc: `option "output" {
				description "old"
				description "new"
				short "x"
				short "o"
			}
			flag "verbose" {
				description "old"
				description "new"
				short "x"
				short "v"
			}
			`,
			options: []v1.Option{
				{
					Flag: v1.Flag{
						Name:        "output",
						Description: "new",
						Short:       "o",
					},
				},
			},
			flags: []v1.Flag{
				{
					Name:        "verbose",
					Description: "new",
					Short:       "v",
				},
			},
		},
	} {
		suite.Run(test.name, func() {
			conf, err := v1.Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)
			suite.Require().NotNil(conf)
			suite.Equal(test.options, conf.Options)
			suite.Equal(test.flags, conf.Flags)
		})
	}
}

func (suite *ParseTestSuite) TestPositionalArguments() {
	for _, test := range []struct {
		name   string
		doc    string
		first  []v1.Arg
		last   []v1.Arg
		vararg *v1.VarArg
	}{
		{
			name: "without vararg",
			doc:  "arg \"source\"\narg \"dest\"\n",
			first: []v1.Arg{
				{
					Name: "source",
				},
				{
					Name: "dest",
				},
			},
		},
		{
			name: "leading and trailing order",
			doc:  "arg \"first\"\narg \"second\"\nvararg \"items\"\narg \"third\"\narg \"fourth\"\n",
			first: []v1.Arg{
				{
					Name: "first",
				},
				{
					Name: "second",
				},
			},
			last: []v1.Arg{
				{
					Name: "third",
				},
				{
					Name: "fourth",
				},
			},
			vararg: &v1.VarArg{
				Arg: v1.Arg{
					Name: "items",
				},
			},
		},
		{
			name: "configured argument",
			doc: `arg "source" {
				description "old"
				description "Source file"
				value "str" {
					re ".*"
				}
			}
			`,
			first: []v1.Arg{
				{
					Name:        "source",
					Description: "Source file",
					WithValue: v1.WithValue{
						Type: "str",
						Properties: map[string][]any{
							"re": {".*"},
						},
					},
				},
			},
		},
		{
			name: "positional names are independent",
			doc:  "arg \"item\"\nvararg \"item\"\narg \"item\"\noption \"item\"\n",
			first: []v1.Arg{
				{
					Name: "item",
				},
			},
			last: []v1.Arg{
				{
					Name: "item",
				},
			},
			vararg: &v1.VarArg{
				Arg: v1.Arg{
					Name: "item",
				},
			},
		},
	} {
		suite.Run(test.name, func() {
			conf, err := v1.Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)
			suite.Require().NotNil(conf)
			suite.Equal(test.first, conf.FirstArgs)
			suite.Equal(test.last, conf.LastArgs)
			suite.Equal(test.vararg, conf.VarArgs)
		})
	}
}

func (suite *ParseTestSuite) TestVarArgCounts() {
	for _, test := range []struct {
		name string
		doc  string
		min  *int64
		max  *int64
	}{
		{
			name: "omitted",
		},
		{
			name: "minimum only",
			doc:  "min-count 1\n",
			min:  new(int64(1)),
		},
		{
			name: "maximum only",
			doc:  "max-count 5\n",
			max:  new(int64(5)),
		},
		{
			name: "zero bounds",
			doc:  "min-count 0\nmax-count 0\n",
			min:  new(int64(0)),
			max:  new(int64(0)),
		},
		{
			name: "equal bounds",
			doc:  "min-count 3\nmax-count 3\n",
			min:  new(int64(3)),
			max:  new(int64(3)),
		},
		{
			name: "negative bounds",
			doc:  "min-count -3\nmax-count -1\n",
			min:  new(int64(-3)),
			max:  new(int64(-1)),
		},
		{
			name: "int64 bounds",
			doc:  "min-count -9223372036854775808\nmax-count 9223372036854775807\n",
			min:  new(int64(-9223372036854775808)),
			max:  new(int64(9223372036854775807)),
		},
		{
			name: "last count wins",
			doc:  "min-count 10\nmax-count 1\nmin-count 2\nmax-count 4\n",
			min:  new(int64(2)),
			max:  new(int64(4)),
		},
	} {
		suite.Run(test.name, func() {
			doc := "vararg \"items\" {\n" + test.doc + "}\n"
			conf, err := v1.Parse(strings.NewReader(doc))
			suite.Require().NoError(err)
			suite.Require().NotNil(conf)
			suite.Equal(&v1.VarArg{
				Arg: v1.Arg{
					Name: "items",
				},
				MinCount: test.min,
				MaxCount: test.max,
			}, conf.VarArgs)
		})
	}
}

func (suite *ParseTestSuite) TestValues() {
	for _, node := range []string{"option", "arg", "vararg"} {
		suite.Run(node, func() {
			for _, test := range []struct {
				name string
				doc  string
				want v1.WithValue
			}{
				{
					name: "omitted",
				},
				{
					name: "without properties",
					doc:  "value \"str\"\n",
					want: v1.WithValue{
						Type:       "str",
						Properties: map[string][]any{},
					},
				},
				{
					name: "unvalidated type and properties",
					doc: `value "custom" {
						mixed "text" 2 1.5 #true #false #null
						empty
					}
					`,
					want: v1.WithValue{
						Type: "custom",
						Properties: map[string][]any{
							"mixed": {"text", int64(2), float64(1.5), true, false, nil},
							"empty": {},
						},
					},
				},
				{
					name: "last property wins",
					doc:  "value \"str\" {\nre \"old\"\nre \"new\" \"other\"\n}\n",
					want: v1.WithValue{
						Type: "str",
						Properties: map[string][]any{
							"re": {"new", "other"},
						},
					},
				},
				{
					name: "value replaces type and properties",
					doc:  "value \"old\" {\nre \"old\"\n}\nvalue \"new\" {\nmin-length 2\n}\n",
					want: v1.WithValue{
						Type: "new",
						Properties: map[string][]any{
							"min-length": {int64(2)},
						},
					},
				},
				{
					name: "value clears properties",
					doc:  "value \"str\" {\nre \"old\"\n}\nvalue \"str\"\n",
					want: v1.WithValue{
						Type:       "str",
						Properties: map[string][]any{},
					},
				},
			} {
				suite.Run(test.name, func() {
					doc := node + " \"item\" {\ndescription \"Item\"\n" + test.doc + "}\n"
					conf, err := v1.Parse(strings.NewReader(doc))
					suite.Require().NoError(err)
					suite.Require().NotNil(conf)
					switch node {
					case "option":
						suite.Require().Len(conf.Options, 1)
						suite.Equal(test.want, conf.Options[0].WithValue)
						suite.Equal("Item", conf.Options[0].Description)
					case "arg":
						suite.Require().Len(conf.FirstArgs, 1)
						suite.Equal(test.want, conf.FirstArgs[0].WithValue)
						suite.Equal("Item", conf.FirstArgs[0].Description)
					case "vararg":
						suite.Require().NotNil(conf.VarArgs)
						suite.Equal(test.want, conf.VarArgs.WithValue)
						suite.Equal("Item", conf.VarArgs.Description)
					}
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestEmptyValueType() {
	for _, node := range []string{"option", "arg", "vararg"} {
		suite.Run(node, func() {
			for _, test := range []struct {
				name string
				doc  string
			}{
				{
					name: "without properties",
					doc:  "value \"\"\n",
				},
				{
					name: "with properties",
					doc:  "value \"\" {\nmin-length 3\n}\n",
				},
				{
					name: "after valid value",
					doc:  "value \"str\"\nvalue \"\"\n",
				},
				{
					name: "before valid value",
					doc:  "value \"\"\nvalue \"str\"\n",
				},
			} {
				suite.Run(test.name, func() {
					doc := node + " \"item\" {\n" + test.doc + "}\n"
					conf, err := v1.Parse(strings.NewReader(doc))
					suite.Require().Error(err)
					suite.ErrorContains(err, "cannot process node "+node+":")
					suite.ErrorContains(err, "please define a value type")
					suite.Nil(conf)
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestErrors() {
	for _, test := range []struct {
		name    string
		doc     string
		message string
		want    error
	}{
		{
			name:    "malformed KDL",
			doc:     "option \"unfinished\" {\n",
			message: "cannot parse config as KDL:",
		},
		{
			name:    "unknown config node",
			doc:     "unknown\n",
			message: "cannot process node unknown:",
			want:    v1.ErrUnknownNode,
		},
		{
			name: "empty execute",
			doc:  "execute\n",
			want: v1.ErrDefineExecute,
		},
		{
			name: "last execute empty",
			doc:  "execute \"sh\"\nexecute\n",
			want: v1.ErrDefineExecute,
		},
		{
			name:    "execute nonstring argument",
			doc:     "execute \"bash\" 1\n",
			message: "cannot process node execute: expected string parameter, got int64",
		},
		{
			name:    "multiple varargs",
			doc:     "vararg \"first\"\nvararg \"second\"\n",
			message: "cannot process node vararg:",
			want:    v1.ErrOneArgumentExpected,
		},
		{
			name:    "reversed bounds",
			doc:     "vararg \"items\" {\nmin-count 3\nmax-count 2\n}\n",
			message: "min-count 3 is greater than max-count 2",
		},
		{
			name:    "unknown option child",
			doc:     "option \"item\" { unknown; }\n",
			message: "cannot process option unknown:",
			want:    v1.ErrUnknownNode,
		},
		{
			name:    "unknown flag child",
			doc:     "flag \"item\" { value \"str\"; }\n",
			message: "cannot process flag value:",
			want:    v1.ErrUnknownNode,
		},
		{
			name:    "unknown argument child",
			doc:     "arg \"item\" { short \"i\"; }\n",
			message: "cannot process argument short:",
			want:    v1.ErrUnknownNode,
		},
		{
			name:    "unknown vararg child",
			doc:     "vararg \"item\" { short \"i\"; }\n",
			message: "cannot process vararg short:",
			want:    v1.ErrUnknownNode,
		},
	} {
		suite.Run(test.name, func() {
			conf, err := v1.Parse(strings.NewReader(test.doc))
			suite.Require().Error(err)
			suite.Nil(conf)
			if test.message != "" {
				suite.ErrorContains(err, test.message)
			}
			if test.want != nil {
				suite.ErrorIs(err, test.want)
			}
		})
	}
}

func (suite *ParseTestSuite) TestDuplicateNames() {
	for _, first := range []string{"option", "flag"} {
		for _, second := range []string{"option", "flag"} {
			suite.Run(first+" then "+second, func() {
				for _, test := range []struct {
					name    string
					first   string
					second  string
					message string
				}{
					{
						name:    "long",
						first:   "\"item\"\n",
						second:  "\"item\"\n",
						message: "duplicate long name item",
					},
					{
						name:    "short",
						first:   "\"first\" { short \"i\"; }\n",
						second:  "\"second\" { short \"i\"; }\n",
						message: "duplicate short name i",
					},
				} {
					suite.Run(test.name, func() {
						doc := first + " " + test.first + second + " " + test.second
						conf, err := v1.Parse(strings.NewReader(doc))
						suite.Require().Error(err)
						suite.ErrorContains(err, test.message)
						suite.Nil(conf)
					})
				}
			})
		}
	}
}

func (suite *ParseTestSuite) TestInvalidScalarArguments() {
	for _, node := range []struct {
		name     string
		prefix   string
		suffix   string
		value    string
		typeName string
	}{
		{
			name:     "description",
			prefix:   "description",
			value:    "\"text\"",
			typeName: "string",
		},
		{
			name:     "example",
			prefix:   "example",
			value:    "\"text\"",
			typeName: "string",
		},
		{
			name:     "option name",
			prefix:   "option",
			value:    "\"item\"",
			typeName: "string",
		},
		{
			name:     "flag name",
			prefix:   "flag",
			value:    "\"item\"",
			typeName: "string",
		},
		{
			name:     "argument name",
			prefix:   "arg",
			value:    "\"item\"",
			typeName: "string",
		},
		{
			name:     "vararg name",
			prefix:   "vararg",
			value:    "\"item\"",
			typeName: "string",
		},
		{
			name:     "option description",
			prefix:   "option \"item\" {\ndescription",
			suffix:   "}\n",
			value:    "\"text\"",
			typeName: "string",
		},
		{
			name:     "flag description",
			prefix:   "flag \"item\" {\ndescription",
			suffix:   "}\n",
			value:    "\"text\"",
			typeName: "string",
		},
		{
			name:     "argument description",
			prefix:   "arg \"item\" {\ndescription",
			suffix:   "}\n",
			value:    "\"text\"",
			typeName: "string",
		},
		{
			name:     "vararg description",
			prefix:   "vararg \"item\" {\ndescription",
			suffix:   "}\n",
			value:    "\"text\"",
			typeName: "string",
		},
		{
			name:     "option short",
			prefix:   "option \"item\" {\nshort",
			suffix:   "}\n",
			value:    "\"i\"",
			typeName: "string",
		},
		{
			name:     "flag short",
			prefix:   "flag \"item\" {\nshort",
			suffix:   "}\n",
			value:    "\"i\"",
			typeName: "string",
		},
		{
			name:     "option value",
			prefix:   "option \"item\" {\nvalue",
			suffix:   "}\n",
			value:    "\"str\"",
			typeName: "string",
		},
		{
			name:     "argument value",
			prefix:   "arg \"item\" {\nvalue",
			suffix:   "}\n",
			value:    "\"str\"",
			typeName: "string",
		},
		{
			name:     "vararg value",
			prefix:   "vararg \"item\" {\nvalue",
			suffix:   "}\n",
			value:    "\"str\"",
			typeName: "string",
		},
		{
			name:     "minimum count",
			prefix:   "vararg \"item\" {\nmin-count",
			suffix:   "}\n",
			value:    "1",
			typeName: "int64",
		},
		{
			name:     "maximum count",
			prefix:   "vararg \"item\" {\nmax-count",
			suffix:   "}\n",
			value:    "1",
			typeName: "int64",
		},
	} {
		suite.Run(node.name, func() {
			for _, test := range []struct {
				name    string
				args    string
				message string
				want    error
			}{
				{
					name: "missing",
					want: utils.ErrEmpty,
				},
				{
					name:    "multiple",
					args:    node.value + " " + node.value,
					message: "expected 1 element, got 2",
				},
				{
					name:    "wrong type",
					args:    "#true",
					message: fmt.Sprintf("expected %s parameter, got bool", node.typeName),
				},
			} {
				suite.Run(test.name, func() {
					doc := node.prefix + " " + test.args + "\n" + node.suffix
					conf, err := v1.Parse(strings.NewReader(doc))
					suite.Require().Error(err)
					suite.Nil(conf)
					if test.message != "" {
						suite.ErrorContains(err, test.message)
					}
					if test.want != nil {
						suite.ErrorIs(err, test.want)
					}
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestInvalidNames() {
	for _, node := range []string{"option", "flag", "arg", "vararg"} {
		suite.Run(node, func() {
			for _, test := range []struct {
				name  string
				value string
			}{
				{
					name: "empty",
				},
				{
					name:  "punctuation",
					value: "---",
				},
				{
					name:  "nonascii",
					value: "привет",
				},
				{
					name:  "invalid prefix",
					value: "=name",
				},
				{
					name:  "invalid middle",
					value: "na=me",
				},
				{
					name:  "invalid suffix",
					value: "name=",
				},
				{
					name:  "spaces",
					value: "name with spaces",
				},
				{
					name:  "mixed ascii and unicode",
					value: "nameпривет",
				},
			} {
				suite.Run(test.name, func() {
					conf, err := v1.Parse(strings.NewReader(fmt.Sprintf("%s %q\n", node, test.value)))
					suite.Require().Error(err)
					suite.ErrorContains(err, "cannot set a name:")
					suite.ErrorContains(err, "does not match regex")
					suite.Nil(conf)
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestNameControlCharacters() {
	for _, node := range []string{"option", "flag", "arg", "vararg"} {
		suite.Run(node, func() {
			for _, test := range []struct {
				name  string
				value string
			}{
				{
					name:  "embedded nul",
					value: `"na\u{0}me"`,
				},
				{
					name:  "trailing nul",
					value: `"name\u{0}"`,
				},
				{
					name:  "embedded newline",
					value: `"na\nme"`,
				},
				{
					name:  "trailing newline",
					value: `"name\n"`,
				},
			} {
				suite.Run(test.name, func() {
					conf, err := v1.Parse(strings.NewReader(node + " " + test.value + "\n"))
					suite.Require().Error(err)
					suite.ErrorContains(err, "cannot set a name:")
					suite.ErrorContains(err, "does not match regex")
					suite.Nil(conf)
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestValidNames() {
	for _, node := range []string{"option", "flag", "arg", "vararg"} {
		suite.Run(node, func() {
			for _, test := range []struct {
				name  string
				value string
			}{
				{
					name:  "lowercase letters",
					value: "output",
				},
				{
					name:  "mixed case and digit",
					value: "Output2",
				},
				{
					name:  "digits only",
					value: "123",
				},
			} {
				suite.Run(test.name, func() {
					conf, err := v1.Parse(strings.NewReader(fmt.Sprintf("%s %q\n", node, test.value)))
					suite.Require().NoError(err)
					suite.Require().NotNil(conf)
					switch node {
					case "option":
						suite.Require().Len(conf.Options, 1)
						suite.Equal(test.value, conf.Options[0].Name)
					case "flag":
						suite.Require().Len(conf.Flags, 1)
						suite.Equal(test.value, conf.Flags[0].Name)
					case "arg":
						suite.Require().Len(conf.FirstArgs, 1)
						suite.Equal(test.value, conf.FirstArgs[0].Name)
					case "vararg":
						suite.Require().NotNil(conf.VarArgs)
						suite.Equal(test.value, conf.VarArgs.Name)
					}
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestInvalidShorts() {
	for _, node := range []string{"option", "flag"} {
		suite.Run(node, func() {
			for _, test := range []struct {
				name  string
				value string
				count int
			}{
				{
					name: "empty",
				},
				{
					name:  "multiple ascii characters",
					value: "ab",
					count: 2,
				},
				{
					name:  "multiple unicode characters",
					value: "привет",
					count: 6,
				},
			} {
				suite.Run(test.name, func() {
					doc := fmt.Sprintf("%s \"item\" { short %q; }\n", node, test.value)
					conf, err := v1.Parse(strings.NewReader(doc))
					suite.Require().Error(err)
					suite.ErrorContains(err, fmt.Sprintf("short must contain 1 character, not %d", test.count))
					suite.Nil(conf)
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestShortNameCharacters() {
	for _, node := range []string{"option", "flag"} {
		suite.Run(node, func() {
			for _, test := range []struct {
				name  string
				value string
				valid bool
			}{
				{
					name:  "lowercase letter",
					value: "a",
					valid: true,
				},
				{
					name:  "uppercase letter",
					value: "Z",
					valid: true,
				},
				{
					name:  "digit",
					value: "0",
					valid: true,
				},
				{
					name:  "unicode letter from привет",
					value: "п",
				},
				{
					name:  "hyphen",
					value: "-",
				},
				{
					name:  "underscore",
					value: "_",
				},
				{
					name:  "equals",
					value: "=",
				},
				{
					name:  "space",
					value: " ",
				},
			} {
				suite.Run(test.name, func() {
					doc := fmt.Sprintf("%s \"item\" { short %q; }\n", node, test.value)
					conf, err := v1.Parse(strings.NewReader(doc))
					if !test.valid {
						suite.Require().Error(err)
						suite.ErrorContains(err, "cannot process "+node+" short:")
						suite.ErrorContains(err, "must comply "+v1.ReName.String()+" regexp")
						suite.Nil(conf)
						return
					}
					suite.Require().NoError(err)
					suite.Require().NotNil(conf)
					if node == "option" {
						suite.Require().Len(conf.Options, 1)
						suite.Equal(test.value, conf.Options[0].Short)
					} else {
						suite.Require().Len(conf.Flags, 1)
						suite.Equal(test.value, conf.Flags[0].Short)
					}
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestReaders() {
	doc := "description \"привет\"\n"
	for _, test := range []struct {
		name  string
		input io.Reader
	}{
		{
			name:  "one byte reads",
			input: iotest.OneByteReader(strings.NewReader(doc)),
		},
		{
			name:  "half reads",
			input: iotest.HalfReader(strings.NewReader(doc)),
		},
		{
			name:  "data and eof together",
			input: iotest.DataErrReader(strings.NewReader(doc)),
		},
	} {
		suite.Run(test.name, func() {
			conf, err := v1.Parse(test.input)
			suite.Require().NoError(err)
			suite.Require().NotNil(conf)
			suite.Equal("привет", conf.Description)
			suite.Equal([]string{"bash"}, conf.Argv)
		})
	}
}

func TestParse(t *testing.T) {
	suite.Run(t, &ParseTestSuite{})
}
