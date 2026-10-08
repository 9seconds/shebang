package log

import (
	"bytes"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/suite"
)

type LogTestSuite struct {
	suite.Suite
	output bytes.Buffer
}

func (suite *LogTestSuite) SetupTest() {
	writer := main.Writer()
	suite.T().Cleanup(func() {
		main.SetOutput(writer)
	})
	suite.output.Reset()
	main.SetOutput(&suite.output)
}

func (suite *LogTestSuite) TestConfigure() {
	for _, value := range []string{"", "1", "false"} {
		suite.Run("value="+value, func() {
			const key = "SHEBANG_LOG_TEST_CONFIGURE"
			suite.T().Setenv(key, value)
			main.SetOutput(io.Discard)

			Configure(key)

			suite.Same(os.Stderr, main.Writer())
			_, exists := os.LookupEnv(key)
			suite.False(exists)
		})
	}
}

func (suite *LogTestSuite) TestConfigureUnsetVariable() {
	const key = "SHEBANG_LOG_TEST_CONFIGURE"
	suite.T().Setenv(key, "1")
	suite.Require().NoError(os.Unsetenv(key))

	Configure(key)

	suite.Same(&suite.output, main.Writer())
}

func (suite *LogTestSuite) TestPrint() {
	Print("hello %s: %d", "world", 42)

	suite.Equal(">>> :  hello world: 42\n", suite.output.String())
}

func (suite *LogTestSuite) TestPrintVal() {
	for _, test := range []struct {
		name   string
		reason string
		value  string
		want   string
	}{
		{name: "reason", reason: "status", value: "ready", want: ">>> status:  ready\n"},
		{name: "reason with colon", reason: "status:", value: "ready", want: ">>> status: ready\n"},
		{name: "empty reason", value: "ready", want: ">>> :  ready\n"},
		{name: "empty value", reason: "status", want: ">>> status:  \n"},
		{name: "whitespace value", reason: "status", value: " \t\n\u2003", want: ">>> status:  \n"},
		{
			name: "trim surrounding whitespace", reason: " \tstatus\u2003 ",
			value: "\u2003 ready \t\n", want: ">>> status:  ready\n",
		},
		{
			name: "multiline alignment", reason: "status", value: "first\nsecond\nthird",
			want: ">>> status:  first\n>>>          second\n>>>          third\n",
		},
		{
			name: "multiline colon alignment", reason: "status:", value: "first\nsecond",
			want: ">>> status: first\n>>>         second\n",
		},
		{
			name: "interior whitespace", reason: "x", value: "first \t\n\n  second \u2003\nthird",
			want: ">>> x:  first\n>>>     \n>>>       second\n>>>     third\n",
		},
	} {
		suite.Run(test.name, func() {
			suite.output.Reset()
			PrintVal(test.reason, test.value)
			suite.Equal(test.want, suite.output.String())
		})
	}
}

func (suite *LogTestSuite) TestIterLines() {
	for _, test := range []struct {
		name  string
		value string
		want  []string
	}{
		{name: "empty", want: []string{""}},
		{name: "whitespace only", value: " \t\n\u2003", want: []string{""}},
		{name: "single line", value: "hello", want: []string{"hello"}},
		{name: "trailing newlines", value: "hello\n\n", want: []string{"hello"}},
		{name: "multiple lines", value: "one\ntwo\nthree", want: []string{"one", "two", "three"}},
		{name: "blank lines preserved", value: "\none\n\ntwo", want: []string{"", "one", "", "two"}},
		{name: "trailing whitespace trimmed", value: "one \t\ntwo\u2003\n", want: []string{"one", "two"}},
		{name: "leading whitespace preserved", value: "  one\n\ttwo", want: []string{"  one", "\ttwo"}},
		{name: "CRLF", value: "one\r\ntwo\r\n", want: []string{"one", "two"}},
	} {
		suite.Run(test.name, func() {
			suite.Equal(test.want, slices.Collect(iterLines(test.value)))
		})
	}
}

func (suite *LogTestSuite) TestIterLinesEarlyStop() {
	var lines []string
	iterLines("one\ntwo\nthree")(func(line string) bool {
		lines = append(lines, line)
		return false
	})

	suite.Equal([]string{"one"}, lines)
}

func TestLog(t *testing.T) {
	suite.Run(t, &LogTestSuite{})
}
