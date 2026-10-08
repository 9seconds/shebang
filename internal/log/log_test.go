package log

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

const logProcessCase = "SHEBANG_TEST_LOG_PROCESS_CASE"

type LogTestSuite struct {
	suite.Suite

	output bytes.Buffer
}

func (suite *LogTestSuite) SetupTest() {
	suite.output.Reset()

	previous := main.Writer()
	main.SetOutput(&suite.output)

	suite.T().Cleanup(func() {
		main.SetOutput(previous)
	})
}

func (suite *LogTestSuite) TestPrintVal() {
	for _, test := range []struct {
		name   string
		reason string
		value  string
		want   string
	}{
		{
			name: "empty value",
			want: ">>>  \n",
		},
		{
			name:  "plain value",
			value: "hello",
			want:  ">>>  hello\n",
		},
		{
			name:   "reason gets colon",
			reason: "Name",
			value:  "value",
			want:   ">>> Name:  value\n",
		},
		{
			name:   "existing colon",
			reason: "Name:",
			value:  "value",
			want:   ">>> Name: value\n",
		},
		{
			name:   "trim outer whitespace",
			reason: " \tName \n",
			value:  " \tvalue\n \t",
			want:   ">>> Name:  value\n",
		},
		{
			name:   "whitespace only",
			reason: " \t\n",
			value:  " \t\n",
			want:   ">>>  \n",
		},
		{
			name:   "empty value with reason",
			reason: "Name",
			want:   ">>> Name:  \n",
		},
		{
			name:   "multiline indentation",
			reason: "Name",
			value:  "first\nsecond\nthird",
			want:   ">>> Name:  first\n>>>        second\n>>>        third\n",
		},
		{
			name:   "multiline with existing colon",
			reason: "Name:",
			value:  "first\nsecond",
			want:   ">>> Name: first\n>>>       second\n",
		},
		{
			name:  "blank lines and inner indentation",
			value: "first  \n\n  second\t\n",
			want:  ">>>  first\n>>>  \n>>>    second\n",
		},
		{
			name:  "windows line endings",
			value: "first\r\nsecond\r\n",
			want:  ">>>  first\n>>>  second\n",
		},
		{
			name:   "unicode value",
			reason: "Greeting",
			value:  "привет\nпривет",
			want:   ">>> Greeting:  привет\n>>>            привет\n",
		},
	} {
		suite.Run(test.name, func() {
			suite.output.Reset()
			PrintVal(test.reason, test.value)
			suite.Equal(test.want, suite.output.String())
		})
	}
}

func (suite *LogTestSuite) TestPrintf() {
	for _, test := range []struct {
		name   string
		format string
		args   []any
		want   string
	}{
		{
			name: "empty format",
			want: ">>>  \n",
		},
		{
			name:   "literal message",
			format: "message",
			want:   ">>>  message\n",
		},
		{
			name:   "formatted arguments",
			format: "%s %d %t",
			args:   []any{"привет", 2, true},
			want:   ">>>  привет 2 true\n",
		},
		{
			name:   "formatted multiline value",
			format: "  %s\n  ",
			args:   []any{"first\nsecond"},
			want:   ">>>  first\n>>>  second\n",
		},
	} {
		suite.Run(test.name, func() {
			suite.output.Reset()
			Printf(test.format, test.args...)
			suite.Equal(test.want, suite.output.String())
		})
	}
}

func (suite *LogTestSuite) TestConfigure() {
	for _, test := range []struct {
		name  string
		debug bool
	}{
		{
			name: "disabled preserves output",
		},
		{
			name:  "enabled uses stderr",
			debug: true,
		},
	} {
		suite.Run(test.name, func() {
			main.SetOutput(&suite.output)
			Configure(test.debug)

			if test.debug {
				suite.Same(os.Stderr, main.Writer())
			} else {
				suite.Same(&suite.output, main.Writer())
			}
		})
	}

	Configure(true)
	Configure(false)
	suite.Same(os.Stderr, main.Writer())
}

func (suite *LogTestSuite) TestIterLines() {
	for _, test := range []struct {
		name  string
		value string
		want  []string
	}{
		{
			name: "empty",
			want: []string{""},
		},
		{
			name:  "whitespace only",
			value: " \t\n",
			want:  []string{""},
		},
		{
			name:  "single line",
			value: "привет",
			want:  []string{"привет"},
		},
		{
			name:  "preserve leading whitespace",
			value: "  first\n\tsecond",
			want:  []string{"  first", "\tsecond"},
		},
		{
			name:  "trim trailing whitespace on each line",
			value: "first \t\nsecond\t\n\n",
			want:  []string{"first", "second"},
		},
		{
			name:  "preserve inner empty lines",
			value: "\nfirst\n\nsecond",
			want:  []string{"", "first", "", "second"},
		},
		{
			name:  "windows line endings",
			value: "first\r\nsecond\r\n",
			want:  []string{"first", "second"},
		},
	} {
		suite.Run(test.name, func() {
			var lines []string
			for line := range iterLines(test.value) {
				lines = append(lines, line)
			}

			suite.Equal(test.want, lines)
		})
	}
}

func (suite *LogTestSuite) TestIterLinesEarlyStop() {
	for _, test := range []struct {
		name  string
		value string
		want  string
	}{
		{
			name: "empty",
		},
		{
			name:  "stop before second line",
			value: "привет\nsecond\nthird",
			want:  "привет",
		},
	} {
		suite.Run(test.name, func() {
			calls := 0

			iterLines(test.value)(func(line string) bool {
				calls++

				suite.Equal(test.want, line)

				return false
			})
			suite.Equal(1, calls)
		})
	}
}

func (suite *LogTestSuite) TestDief() {
	executable, executableError := os.Executable()
	suite.Require().NoError(executableError)

	for _, test := range []struct {
		name string
		mode string
		want string
	}{
		{
			name: "formatted message",
			mode: "formatted",
			want: "failed привет: 3\n",
		},
		{
			name: "trailing whitespace",
			mode: "whitespace",
			want: "failed\n",
		},
		{
			name: "empty message",
			mode: "empty",
			want: "\n",
		},
		{
			name: "multiline message",
			mode: "multiline",
			want: "first\nsecond\n",
		},
	} {
		suite.Run(test.name, func() {
			cmd := exec.Command(executable, "-test.run=^TestLogProcess$")

			for _, entry := range os.Environ() {
				name, _, _ := strings.Cut(entry, "=")
				if name != logProcessCase {
					cmd.Env = append(cmd.Env, entry)
				}
			}

			cmd.Env = append(cmd.Env, logProcessCase+"="+test.mode)

			var stderr bytes.Buffer

			cmd.Stderr = &stderr
			cmd.Stdout = io.Discard
			processError := cmd.Run()

			var exitError *exec.ExitError
			suite.Require().ErrorAs(processError, &exitError)
			suite.Equal(1, exitError.ExitCode())
			suite.Equal(test.want, stderr.String())
		})
	}
}

//nolint:paralleltest // The subprocess helper verifies logging that terminates the process.
func TestLogProcess(t *testing.T) {
	switch os.Getenv(logProcessCase) {
	case "":
		t.Skip("subprocess helper")
	case "formatted":
		Dief("failed %s: %d", "привет", 3)
	case "whitespace":
		Dief("failed \t\r\n  ")
	case "empty":
		Dief("")
	case "multiline":
		Dief("first\nsecond\n\n")
	default:
		t.Fatal("unknown subprocess mode")
	}

	t.Fatal("Dief returned instead of exiting")
}

//nolint:paralleltest // The suite mutates the shared logger output.
func TestLog(t *testing.T) {
	suite.Run(t, &LogTestSuite{})
}
