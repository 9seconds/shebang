package env_test

import (
	"os"
	"testing"

	"github.com/9seconds/shebang/internal/env"
	"github.com/stretchr/testify/suite"
)

type EnvTestSuite struct {
	suite.Suite
}

func (suite *EnvTestSuite) TestVar() {
	for _, test := range []struct {
		name string
		arg  string
		want string
	}{
		{name: "empty name", want: "SHEBANG_"},
		{name: "lowercase name", arg: "verbose", want: "SHEBANG_VERBOSE"},
		{name: "uppercase name", arg: "DEBUG", want: "SHEBANG_DEBUG"},
		{name: "mixed case name", arg: "OutputFile", want: "SHEBANG_OUTPUTFILE"},
		{name: "existing prefix", arg: "SHEBANG_VERBOSE", want: "SHEBANG_VERBOSE"},
	} {
		suite.Run(test.name, func() {
			suite.Equal(test.want, env.Var(test.arg))
		})
	}
}

func (suite *EnvTestSuite) TestSet() {
	for _, test := range []struct {
		name  string
		arg   string
		key   string
		value string
	}{
		{name: "unprefixed name", arg: "env_test", key: "SHEBANG_ENV_TEST", value: "new value"},
		{name: "prefixed name", arg: "SHEBANG_ENV_TEST", key: "SHEBANG_ENV_TEST", value: "new value"},
		{name: "empty value", arg: "env_test", key: "SHEBANG_ENV_TEST"},
		{name: "empty name", key: "SHEBANG_", value: "new value"},
		{name: "value preserved", arg: "env_test", key: "SHEBANG_ENV_TEST", value: " привет=world\n "},
	} {
		suite.Run(test.name, func() {
			suite.T().Setenv(test.key, "previous value")
			env.Set(test.arg, test.value)
			value, exists := os.LookupEnv(test.key)
			suite.True(exists)
			suite.Equal(test.value, value)
		})
	}
}

func (suite *EnvTestSuite) TestSetUnsetVariable() {
	const key = "SHEBANG_ENV_TEST"
	suite.T().Setenv(key, "previous value")
	suite.Require().NoError(os.Unsetenv(key))

	env.Set("env_test", "new value")
	value, exists := os.LookupEnv(key)
	suite.True(exists)
	suite.Equal("new value", value)
}

func TestEnv(t *testing.T) {
	suite.Run(t, &EnvTestSuite{})
}
