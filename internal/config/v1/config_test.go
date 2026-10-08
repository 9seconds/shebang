package v1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
}

func (suite *ConfigTestSuite) parse(doc string) *Config {
	conf, err := Parse(strings.NewReader(doc))
	suite.Require().NoError(err)
	return conf
}

func (suite *ConfigTestSuite) TestValueParameterLists() {
	for _, node := range []string{`argument`, `option "name"`} {
		suite.Run(node, func() {
			conf := suite.parse(node + ` {
				value "str" {
					empty
					single 3
					multiple 1 "two" #true 1.5
				}
			}`)
			item := conf.argument.configItem

			if strings.HasPrefix(node, "option") {
				item = conf.options["name"].configItem
			}

			suite.Equal("str", item.valueType)
			suite.Equal(map[string][]any{
				"empty":    {},
				"single":   {int64(3)},
				"multiple": {int64(1), "two", true, 1.5},
			}, item.valueParams)
		})
	}
}

func (suite *ConfigTestSuite) TestRepeatedValueParameters() {
	conf := suite.parse(`argument {
		value "str" {
			min-length 1
			min-length 3 4
		}
	}`)

	suite.Equal([]any{int64(3), int64(4)}, conf.argument.valueParams["min-length"])
}

func (suite *ConfigTestSuite) TestRepeatedValueNode() {
	conf := suite.parse(`argument {
		value "int" { min 1; }
		value "str" { re "hello"; }
	}`)

	suite.Equal("str", conf.argument.valueType)
	suite.Equal(map[string][]any{"re": {"hello"}}, conf.argument.valueParams)
}

func (suite *ConfigTestSuite) TestConfigureCommand() {
	dir := suite.T().TempDir()
	envPath := filepath.Join(dir, "env")

	suite.Require().NoError(os.WriteFile(envPath, []byte("#!/bin/sh\n"), 0o755))
	suite.T().Setenv("PATH", dir)

	for _, test := range []struct {
		name string
		doc  string
		want []string
	}{
		{name: "default", want: []string{envPath, "-S bash"}},
		{name: "empty execute", doc: "execute", want: []string{envPath, "-S bash"}},
		{
			name: "relative execute", doc: `execute "python3" "-u"`,
			want: []string{envPath, "-S python3 -u"},
		},
		{name: "absolute execute", doc: `execute "/bin/sh" "-e"`, want: []string{"/bin/sh", "-e"}},
	} {
		suite.Run(test.name, func() {
			conf := suite.parse("description \"details\"\nexample \"example usage\"\n" + test.doc)
			cmd := &cli.Command{}

			suite.Require().NoError(conf.Configure(cmd))
			suite.Equal("details", cmd.Long)
			suite.Equal("example usage", cmd.Example)
			suite.Equal(test.want, cmd.Argv)
			suite.Empty(cmd.Options)
		})
	}
}

func (suite *ConfigTestSuite) TestExecuteArgumentsAreCloned() {
	conf := suite.parse(`execute "/bin/sh" "-e"`)

	first, second := &cli.Command{}, &cli.Command{}
	suite.Require().NoError(conf.Configure(first))

	first.Argv[1] = "changed"
	suite.Require().NoError(conf.Configure(second))

	suite.Equal([]string{"/bin/sh", "-e"}, second.Argv)
}

func (suite *ConfigTestSuite) TestMissingEnvExecutable() {
	suite.T().Setenv("PATH", suite.T().TempDir())

	conf := suite.parse("")
	err := conf.Configure(&cli.Command{})
	suite.Require().Error(err)
	suite.ErrorContains(err, "cannot find 'env' in PATH:")

	conf = suite.parse(`execute "/bin/sh"`)
	suite.NoError(conf.Configure(&cli.Command{}))
}

func (suite *ConfigTestSuite) TestConfigureOptions() {
	conf := suite.parse(`execute "/bin/sh"
		option "name" short="n" {
			description "A name"
			min-count 1
			max-count 2
			value "str" { min-length 2; max-length 4; re "^[a-z]+$"; }
		}
		option "other" { value "str"; }
	`)

	cmd := &cli.Command{}
	suite.Require().NoError(conf.Configure(cmd))
	suite.Len(cmd.Options, 2)

	flag := cmd.Flags().Lookup("name")
	suite.Require().NotNil(flag)
	suite.Equal("n", flag.Shorthand)
	suite.Equal("A name", flag.Usage)

	option, ok := flag.Value.(*cli.Option)
	suite.Require().True(ok)
	suite.Contains(cmd.Options, option)
	suite.Equal("name", option.Name)
	suite.Equal("str", option.Type())
	suite.Equal(1, option.MinCount)
	suite.Equal(2, option.MaxCount)
	suite.NoError(option.Validator.Validate("abc"))
	suite.EqualError(option.Validator.Validate("a"), "min-length must have at least 2 characters")
	suite.EqualError(option.Validator.Validate("abcde"),
		"max-length must have at most 4 characters")
	suite.EqualError(option.Validator.Validate("12"), "re does not match ^[a-z]+$")

	other := cmd.Flags().Lookup("other")
	suite.Require().NotNil(other)
	suite.Empty(other.Shorthand)

	otherOption, ok := other.Value.(*cli.Option)
	suite.Require().True(ok)
	suite.Equal(-1, otherOption.MinCount)
	suite.Equal(int(^uint(0)>>1), otherOption.MaxCount)
	suite.NoError(otherOption.Validator.Validate(""))
}

func (suite *ConfigTestSuite) TestConfigureInvalidValidator() {
	for _, test := range []struct {
		name  string
		value string
		want  string
	}{
		{
			name: "missing parameter", value: `value "str" { min-length; }`,
			want: "1 value of int64 must be defined",
		},
		{
			name: "extra parameter", value: `value "str" { min-length 0 1; }`,
			want: "expected 1 parameter of int64 but got 2",
		},
		{
			name: "wrong parameter type", value: `value "str" { min-length "1"; }`,
			want: "expected int64 parameter, but got string",
		},
		{name: "unknown type", value: `value "unknown"`, want: "unknown validator type unknown"},
		{name: "unknown check", value: `value "str" { unknown 1; }`, want: "unknown validator unknown"},
	} {
		suite.Run(test.name, func() {
			conf := suite.parse("execute \"/bin/sh\"\noption \"name\" { " + test.value + "; }")
			err := conf.Configure(&cli.Command{})

			suite.EqualError(err, "cannot initialize validator for name: "+test.want)
		})
	}
}

func TestConfig(t *testing.T) {
	suite.Run(t, &ConfigTestSuite{})
}
