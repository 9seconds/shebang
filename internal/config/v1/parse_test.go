package v1_test

import (
	"fmt"
	"io"
	"math"
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
			name:        "description",
			doc:         "description \"A script\"\n",
			description: "A script",
			argv:        []string{"bash"},
		},
		{
			name: "execute arguments",
			doc:  "execute \"bash\" \"-eu\" \"-o\" \"pipefail\"\n",
			argv: []string{"bash", "-eu", "-o", "pipefail"},
		},
		{
			name: "last scalar wins",
			doc: "description \"old\"\ndescription \"new\"\n" +
				"execute \"bash\" \"-x\"\nexecute \"sh\"\n",
			description: "new",
			argv:        []string{"sh"},
		},
		{
			name: "empty metadata",
			doc:  "description \"\"\n",
			argv: []string{"bash"},
		},
		{
			name:        "unicode metadata",
			doc:         "description \"привет\"\n",
			description: "привет",
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
					Name: "output",
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
					Name:        "output",
					Description: "Output file",
					Short:       "o",
					Type:        "str",
					Properties: map[string][]any{
						"min-length": {int64(1)},
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
					Name:  "b",
					Short: "b",
				},
				{
					Name: "a",
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
					Name:        "output",
					Description: "new",
					Short:       "o",
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
				Name: "items",
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
					Type:        "str",
					Properties: map[string][]any{
						"re": {".*"},
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
				Name: "item",
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
		},
		{
			name: "int64 bounds",
			doc:  fmt.Sprintf("min-count %d\nmax-count %d\n", int64(math.MinInt64), int64(math.MinInt64)),
		},
		{
			name: "required items with unlimited maximum",
			doc:  "min-count 2\nmax-count -1\n",
			min:  new(int64(2)),
		},
		{
			name: "negative minimum with finite maximum",
			doc:  "min-count -3\nmax-count 2\n",
			max:  new(int64(2)),
		},
		{
			name: "negative minimum with zero maximum",
			doc:  "min-count -3\nmax-count 0\n",
			max:  new(int64(0)),
		},
		{
			name: "reversed negative bounds are unlimited",
			doc:  "min-count -1\nmax-count -3\n",
		},
		{
			name: "last negative bounds clear limits",
			doc:  "min-count 3\nmax-count 2\nmin-count -1\nmax-count -1\n",
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
				Name:     "items",
				MinCount: test.min,
				MaxCount: test.max,
			}, conf.VarArgs)
		})
	}
}

func (suite *ParseTestSuite) TestVarArgBoundsLimit() {
	for _, test := range []struct {
		name    string
		doc     string
		min     *int64
		max     *int64
		message string
		bound   string
		count   int64
	}{
		{
			name: "below limit",
			doc:  fmt.Sprintf("min-count %d\n", v1.MaxVarArgs-1),
			min:  new(int64(v1.MaxVarArgs - 1)),
		},
		{
			name: "at limit",
			doc:  fmt.Sprintf("min-count %d\n", v1.MaxVarArgs),
			min:  new(int64(v1.MaxVarArgs)),
		},
		{
			name: "equal bounds at limit",
			doc:  fmt.Sprintf("min-count %d\nmax-count %d\n", v1.MaxVarArgs, v1.MaxVarArgs),
			min:  new(int64(v1.MaxVarArgs)),
			max:  new(int64(v1.MaxVarArgs)),
		},
		{
			name:    "above limit with unlimited maximum",
			bound:   "min-count",
			count:   v1.MaxVarArgs + 1,
			doc:     fmt.Sprintf("min-count %d\nmax-count -1\n", v1.MaxVarArgs+1),
			message: fmt.Sprintf("if you use more than %d max arguments, do not limit them", v1.MaxVarArgs),
		},
		{
			name:    "above limit with finite maximum",
			bound:   "min-count",
			count:   v1.MaxVarArgs + 1,
			doc:     fmt.Sprintf("min-count %d\nmax-count %d\n", v1.MaxVarArgs+1, v1.MaxVarArgs+2),
			message: fmt.Sprintf("if you use more than %d max arguments, do not limit them", v1.MaxVarArgs),
		},
		{
			name:    "largest int64 minimum",
			bound:   "min-count",
			count:   math.MaxInt64,
			doc:     fmt.Sprintf("min-count %d\n", int64(math.MaxInt64)),
			message: fmt.Sprintf("if you use more than %d max arguments, do not limit them", v1.MaxVarArgs),
		},
		{
			name: "maximum below limit without minimum",
			doc:  fmt.Sprintf("max-count %d\n", v1.MaxVarArgs-1),
			max:  new(int64(v1.MaxVarArgs - 1)),
		},
		{
			name: "maximum at limit without minimum",
			doc:  fmt.Sprintf("max-count %d\n", v1.MaxVarArgs),
			max:  new(int64(v1.MaxVarArgs)),
		},
		{
			name:    "maximum above limit without minimum",
			bound:   "max-count",
			count:   v1.MaxVarArgs + 1,
			doc:     fmt.Sprintf("max-count %d\n", v1.MaxVarArgs+1),
			message: fmt.Sprintf("if you use more than %d max arguments, do not limit them", v1.MaxVarArgs),
		},
		{
			name:    "maximum above limit with valid minimum",
			bound:   "max-count",
			count:   v1.MaxVarArgs + 1,
			doc:     fmt.Sprintf("min-count 1\nmax-count %d\n", v1.MaxVarArgs+1),
			message: fmt.Sprintf("if you use more than %d max arguments, do not limit them", v1.MaxVarArgs),
		},
		{
			name:    "largest int64 maximum",
			bound:   "max-count",
			count:   math.MaxInt64,
			doc:     fmt.Sprintf("max-count %d\n", int64(math.MaxInt64)),
			message: fmt.Sprintf("if you use more than %d max arguments, do not limit them", v1.MaxVarArgs),
		},
		{
			name: "unlimited maximum with minimum at limit",
			doc:  fmt.Sprintf("min-count %d\nmax-count -1\n", v1.MaxVarArgs),
			min:  new(int64(v1.MaxVarArgs)),
		},
		{
			name:    "negative minimum does not bypass maximum limit",
			bound:   "max-count",
			count:   v1.MaxVarArgs + 1,
			doc:     fmt.Sprintf("min-count -1\nmax-count %d\n", v1.MaxVarArgs+1),
			message: fmt.Sprintf("if you use more than %d max arguments, do not limit them", v1.MaxVarArgs),
		},
	} {
		suite.Run(test.name, func() {
			conf, err := v1.Parse(strings.NewReader("vararg \"items\" {\n" + test.doc + "}\n"))
			if test.message != "" {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, "cannot process node vararg:")
				suite.Require().ErrorContains(err, test.message)
				suite.Nil(conf)

				var limitError *v1.VarArgLimitError

				suite.Require().ErrorAs(err, &limitError)
				suite.Equal(test.bound, limitError.Bound)
				suite.Equal(test.count, limitError.Count)
				suite.Equal(*v1.NewVarArgLimitError(test.bound, test.count), *limitError)

				return
			}

			suite.Require().NoError(err)
			suite.Require().NotNil(conf)
			suite.Require().NotNil(conf.VarArgs)
			suite.Equal(test.min, conf.VarArgs.MinCount)
			suite.Equal(test.max, conf.VarArgs.MaxCount)
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
					suite.Require().ErrorContains(err, "cannot process node "+node+":")
					suite.Require().ErrorIs(err, v1.ErrNoValueType)
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
			name:    "removed example node",
			doc:     "example \"script source dest\"\n",
			message: "cannot process node example:",
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
				suite.Require().ErrorContains(err, test.message)
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
					want    error
				}{
					{
						name:    "long",
						first:   "\"item\"\n",
						second:  "\"item\"\n",
						message: "duplicate long name item",
						want:    v1.ErrDuplicateLongName,
					},
					{
						name:    "short",
						first:   "\"first\" { short \"i\"; }\n",
						second:  "\"second\" { short \"i\"; }\n",
						message: "duplicate short name i",
						want:    v1.ErrDuplicateShortName,
					},
				} {
					suite.Run(test.name, func() {
						doc := first + " " + test.first + second + " " + test.second
						conf, err := v1.Parse(strings.NewReader(doc))
						suite.Require().Error(err)
						suite.Require().ErrorContains(err, test.message)
						suite.Require().ErrorIs(err, test.want)
						suite.Nil(conf)
					})
				}
			})
		}
	}
}

func (suite *ParseTestSuite) TestCaseInsensitiveDuplicateNames() {
	for _, first := range []string{"option", "flag"} {
		for _, second := range []string{"option", "flag"} {
			suite.Run(first+" then "+second, func() {
				for _, test := range []struct {
					name    string
					first   string
					second  string
					message string
					want    error
				}{
					{
						name:    "uppercase long name first",
						first:   "\"OUTPUT\"\n",
						second:  "\"output\"\n",
						message: "duplicate long name output",
						want:    v1.ErrDuplicateLongName,
					},
					{
						name:    "mixed case long name second",
						first:   "\"output\"\n",
						second:  "\"OuTpUt\"\n",
						message: "duplicate long name output",
						want:    v1.ErrDuplicateLongName,
					},
					{
						name:    "uppercase short name first",
						first:   "\"first\" { short \"O\"; }\n",
						second:  "\"second\" { short \"o\"; }\n",
						message: "duplicate short name o",
						want:    v1.ErrDuplicateShortName,
					},
					{
						name:    "uppercase short name second",
						first:   "\"first\" { short \"o\"; }\n",
						second:  "\"second\" { short \"O\"; }\n",
						message: "duplicate short name o",
						want:    v1.ErrDuplicateShortName,
					},
				} {
					suite.Run(test.name, func() {
						doc := first + " " + test.first + second + " " + test.second
						conf, err := v1.Parse(strings.NewReader(doc))
						suite.Require().Error(err)
						suite.Require().ErrorContains(err, test.message)
						suite.Require().ErrorIs(err, test.want)
						suite.Nil(conf)
					})
				}
			})
		}
	}
}

func (suite *ParseTestSuite) TestReservedNames() {
	for _, node := range []string{"option", "flag", "arg", "vararg"} {
		suite.Run(node, func() {
			for _, test := range []struct {
				name     string
				value    string
				reserved bool
			}{
				{
					name:     "lowercase reserved name",
					value:    "help",
					reserved: true,
				},
				{
					name:     "uppercase reserved name",
					value:    "HELP",
					reserved: true,
				},
				{
					name:     "mixed case reserved name",
					value:    "HeLp",
					reserved: true,
				},
				{
					name:  "reserved name prefix allowed",
					value: "helper",
				},
				{
					name:  "reserved name suffix allowed",
					value: "myhelp",
				},
			} {
				suite.Run(test.name, func() {
					conf, err := v1.Parse(strings.NewReader(fmt.Sprintf("%s %q\n", node, test.value)))
					if test.reserved {
						suite.Require().Error(err)
						suite.Require().ErrorIs(err, v1.ErrReservedName)
						suite.Require().ErrorContains(err, "cannot process node "+node+": cannot set a name:")
						suite.Nil(conf)
					} else {
						suite.Require().NoError(err)
						suite.NotNil(conf)
					}
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestReservedShorts() {
	for _, node := range []string{"option", "flag"} {
		suite.Run(node, func() {
			for _, test := range []struct {
				name     string
				value    string
				reserved bool
			}{
				{
					name:     "lowercase reserved shorthand",
					value:    "h",
					reserved: true,
				},
				{
					name:     "uppercase reserved shorthand",
					value:    "H",
					reserved: true,
				},
				{
					name:  "nonreserved shorthand",
					value: "o",
				},
			} {
				suite.Run(test.name, func() {
					doc := fmt.Sprintf("%s \"item\" { short %q; }\n", node, test.value)

					conf, err := v1.Parse(strings.NewReader(doc))
					if test.reserved {
						suite.Require().Error(err)
						suite.Require().ErrorIs(err, v1.ErrReservedShort)
						suite.Require().ErrorContains(err, "cannot process "+node+" short:")
						suite.Nil(conf)
					} else {
						suite.Require().NoError(err)
						suite.NotNil(conf)
					}
				})
			}
		})
	}
}

func (suite *ParseTestSuite) TestShortNameNormalization() {
	for _, node := range []string{"option", "flag"} {
		suite.Run(node, func() {
			conf, err := v1.Parse(strings.NewReader(node + " \"OuTpUt\" { short \"O\"; }\n"))
			suite.Require().NoError(err)
			suite.Require().NotNil(conf)

			if node == "option" {
				suite.Require().Len(conf.Options, 1)
				suite.Equal("output", conf.Options[0].Name)
				suite.Equal("o", conf.Options[0].Short)
			} else {
				suite.Require().Len(conf.Flags, 1)
				suite.Equal("output", conf.Flags[0].Name)
				suite.Equal("o", conf.Flags[0].Short)
			}
		})
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
						suite.Require().ErrorContains(err, test.message)
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
					name:  "leading hyphen",
					value: "-name",
				},
				{
					name:  "trailing hyphen",
					value: "name-",
				},
				{
					name:  "consecutive hyphens",
					value: "archive--name",
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
					suite.Require().ErrorContains(err, "cannot set a name:")
					suite.Require().ErrorContains(err, "does not match regex")
					suite.Nil(conf)

					var patternError *v1.NamePatternError

					suite.Require().ErrorAs(err, &patternError)
					suite.Equal(strings.ToLower(test.value), patternError.Value)
					suite.Equal(v1.ReName.String(), patternError.Pattern)
					constructed := v1.NewNamePatternError(patternError.Value, patternError.Pattern)
					suite.Equal(*constructed, *patternError)
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
					suite.Require().ErrorContains(err, "cannot set a name:")
					suite.Require().ErrorContains(err, "does not match regex")
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
				{
					name:  "hyphenated name",
					value: "archive-name",
				},
				{
					name:  "multiple hyphens and mixed case",
					value: "Archive-Name-2",
				},
			} {
				suite.Run(test.name, func() {
					conf, err := v1.Parse(strings.NewReader(fmt.Sprintf("%s %q\n", node, test.value)))
					suite.Require().NoError(err)
					suite.Require().NotNil(conf)

					switch node {
					case "option":
						suite.Require().Len(conf.Options, 1)
						suite.Equal(strings.ToLower(test.value), conf.Options[0].Name)
					case "flag":
						suite.Require().Len(conf.Flags, 1)
						suite.Equal(strings.ToLower(test.value), conf.Flags[0].Name)
					case "arg":
						suite.Require().Len(conf.FirstArgs, 1)
						suite.Equal(strings.ReplaceAll(strings.ToLower(test.value), "-", "_"),
							conf.FirstArgs[0].Name)
					case "vararg":
						suite.Require().NotNil(conf.VarArgs)
						suite.Equal(strings.ReplaceAll(strings.ToLower(test.value), "-", "_"),
							conf.VarArgs.Name)
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
					suite.Require().ErrorContains(err,
						fmt.Sprintf("short must contain 1 character, not %d", test.count))
					suite.Nil(conf)

					var lengthError *v1.ShortNameLengthError

					suite.Require().ErrorAs(err, &lengthError)
					suite.Equal(test.count, lengthError.Count)

					constructed := v1.NewShortNameLengthError(test.count)
					suite.Require().NotNil(constructed)
					suite.Equal(*constructed, *lengthError)
					suite.Equal(constructed.Error(), lengthError.Error())
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
						suite.Require().ErrorContains(err, "cannot process "+node+" short:")
						suite.Require().ErrorContains(err, "must comply "+v1.ReShort.String()+" regexp")
						suite.Nil(conf)

						var patternError *v1.ShortNamePatternError

						suite.Require().ErrorAs(err, &patternError)
						suite.Equal(v1.ReShort.String(), patternError.Pattern)
						constructed := v1.NewShortNamePatternError(v1.ReShort.String())
						suite.Equal(*constructed, *patternError)
						suite.Equal(constructed.Error(), patternError.Error())

						return
					}

					suite.Require().NoError(err)
					suite.Require().NotNil(conf)

					if node == "option" {
						suite.Require().Len(conf.Options, 1)
						suite.Equal(strings.ToLower(test.value), conf.Options[0].Short)
					} else {
						suite.Require().Len(conf.Flags, 1)
						suite.Equal(strings.ToLower(test.value), conf.Flags[0].Short)
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
	t.Parallel()

	suite.Run(t, &ParseTestSuite{})
}
