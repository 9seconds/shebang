package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/9seconds/shebang/internal/config/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

const mainProcessMode = "SHEBANG_TEST_MAIN_PROCESS"

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) TestParseConfig() {
	for _, test := range []struct {
		name    string
		doc     string
		missing bool
		message string
	}{
		{
			name: "default configuration",
			doc:  "#!/bin/sh\necho привет\n",
		},
		{
			name: "embedded configuration",
			doc:  "#!shebang.1\n# description \"привет\"\n",
		},
		{
			name:    "missing file",
			missing: true,
			message: "cannot open",
		},
		{
			name:    "invalid configuration",
			doc:     "#!shebang.1\n# unknown\n",
			message: "cannot process node unknown:",
		},
	} {
		suite.Run(test.name, func() {
			path := filepath.Join(suite.T().TempDir(), "script")
			if !test.missing {
				suite.Require().NoError(os.WriteFile(path, []byte(test.doc), 0o600))
			}

			conf, err := parseConfig(path)

			if test.message != "" {
				suite.Require().ErrorContains(err, test.message)
				suite.Nil(conf)

				if test.missing {
					suite.Require().ErrorIs(err, os.ErrNotExist)
				}
			} else {
				suite.Require().NoError(err)
				suite.Require().IsType(&v1.Config{}, conf)

				parsed, ok := conf.(*v1.Config)
				suite.Require().True(ok)
				suite.Equal([]string{"bash"}, parsed.Argv)

				if test.name == "embedded configuration" {
					suite.Equal("привет", parsed.Description)
				}
			}
		})
	}
}

func (suite *MainTestSuite) TestGetScript() {
	for _, test := range []struct {
		name     string
		mode     os.FileMode
		missing  bool
		relative bool
	}{
		{
			name: "absolute executable",
			mode: 0o700,
		},
		{
			name:     "relative executable",
			mode:     0o700,
			relative: true,
		},
		{
			name: "nonexecutable script",
			mode: 0o600,
		},
		{
			name:    "missing script",
			missing: true,
		},
	} {
		suite.Run(test.name, func() {
			path := filepath.Join(suite.T().TempDir(), "script")
			if !test.missing {
				suite.Require().NoError(os.WriteFile(path, []byte("#!/bin/sh\n"), test.mode))
			}

			input := path

			if test.relative {
				cwd, err := os.Getwd()
				suite.Require().NoError(err)
				input, err = filepath.Rel(cwd, path)
				suite.Require().NoError(err)
			}

			previous := os.Args
			os.Args = []string{"shebang", input}

			defer func() { os.Args = previous }()

			got, err := getScript()
			if test.missing || test.mode&0o111 == 0 {
				suite.Require().Error(err)
				suite.Empty(got)
			} else {
				suite.Require().NoError(err)
				suite.Equal(path, got)
			}
		})
	}
}

func (suite *MainTestSuite) TestRunCompletion() {
	for _, test := range []struct {
		name     string
		shell    string
		fallback string
		marker   string
	}{
		{
			name:   "bash",
			shell:  "bash",
			marker: "bash completion",
		},
		{
			name:   "zsh path",
			shell:  "/usr/bin/zsh",
			marker: "#compdef script",
		},
		{
			name:   "fish",
			shell:  "fish",
			marker: "fish completion",
		},
		{
			name:   "pwsh",
			shell:  "pwsh",
			marker: "Register-ArgumentCompleter",
		},
		{
			name:   "pwsh path",
			shell:  "/usr/bin/pwsh",
			marker: "Register-ArgumentCompleter",
		},
		{
			name:     "auto uses shell environment",
			shell:    "auto",
			fallback: "/bin/zsh",
			marker:   "#compdef script",
		},
		{
			name:     "auto detects pwsh",
			shell:    "auto",
			fallback: "/usr/bin/pwsh",
			marker:   "Register-ArgumentCompleter",
		},
		{
			name:     "empty uses shell environment",
			fallback: "/bin/bash",
			marker:   "bash completion",
		},
	} {
		suite.Run(test.name, func() {
			suite.T().Setenv("SHEBANG_COMPLETION", test.shell)
			suite.T().Setenv("SHELL", test.fallback)

			cmd := &cobra.Command{
				Use: "script",
			}

			var output bytes.Buffer

			cmd.SetOut(&output)
			suite.Require().NoError(runCompletion(cmd))
			suite.Contains(output.String(), test.marker)
			suite.Contains(output.String(), "script")
		})
	}
}

func (suite *MainTestSuite) TestUnsupportedCompletionShells() {
	for _, test := range []struct {
		name     string
		shell    string
		fallback string
		want     string
	}{
		{
			name:     "unsupported explicit shell does not use fallback",
			shell:    "unsupported",
			fallback: "/bin/bash",
			want:     "unsupported shell unsupported",
		},
		{
			name:     "unsupported automatically detected shell",
			shell:    "auto",
			fallback: "/bin/unsupported",
			want:     "unsupported shell /bin/unsupported",
		},
		{
			name:  "auto without shell",
			shell: "auto",
			want:  "unsupported shell ",
		},
		{
			name: "empty completion without shell",
			want: "unsupported shell ",
		},
		{
			name:  "removed power alias",
			shell: "power",
			want:  "unsupported shell power",
		},
		{
			name:  "removed powershell alias",
			shell: "powershell",
			want:  "unsupported shell powershell",
		},
	} {
		suite.Run(test.name, func() {
			suite.T().Setenv("SHEBANG_COMPLETION", test.shell)
			suite.T().Setenv("SHELL", test.fallback)

			cmd := &cobra.Command{
				Use: "script",
			}

			var output bytes.Buffer

			cmd.SetOut(&output)
			err := runCompletion(cmd)
			suite.Require().ErrorIs(err, errUnsupportedShell)
			suite.Require().EqualError(err, test.want)
			suite.Empty(output.String())
		})
	}
}

func (suite *MainTestSuite) TestCompletionWriterErrors() {
	want := errors.New("write failed")

	for _, shell := range []string{"bash", "zsh", "fish", "pwsh"} {
		suite.Run(shell, func() {
			suite.T().Setenv("SHEBANG_COMPLETION", shell)

			cmd := &cobra.Command{
				Use: "script",
			}
			cmd.SetOut(mainErrorWriter{err: want})
			suite.ErrorIs(runCompletion(cmd), want)
		})
	}
}

func (suite *MainTestSuite) TestRunDebug() {
	suite.T().Setenv("SHEBANG_TEST_VALUE", "привет=world")
	suite.T().Setenv("OTHER_TEST_VALUE", "do not print")

	var output bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	suite.Require().NoError(runDebug(cmd, []string{"runner", "script", "привет"}))
	suite.Contains(output.String(), "Argv: [runner script привет]\nEnvironment:\n")
	suite.Contains(output.String(), "SHEBANG_TEST_VALUE=привет=world\n")
	suite.NotContains(output.String(), "OTHER_TEST_VALUE")
}

func (suite *MainTestSuite) TestRunExecveError() {
	path := filepath.Join(suite.T().TempDir(), "missing")
	suite.Require().ErrorIs(runExecve(&cobra.Command{}, []string{path}), os.ErrNotExist)
}

type mainTestCase struct {
	name       string
	doc        string
	args       []string
	environ    []string
	noScript   bool
	missing    bool
	exit       int
	stdout     string
	stderr     string
	completion bool
	debug      bool
}

func (suite *MainTestSuite) TestMain() {
	shellPath, err := exec.LookPath("sh")
	suite.Require().NoError(err)

	for _, test := range []mainTestCase{
		{
			name:     "usage without script",
			noScript: true,
			exit:     1,
			stderr:   "usage: shebang <script> [arg...]",
		},
		{
			name:    "missing script",
			missing: true,
			exit:    1,
			stderr:  "cannot detect a script",
		},
		{
			name:   "invalid configuration",
			doc:    "# unknown\n",
			exit:   1,
			stderr: "cannot read a config",
		},
		{
			name:   "invalid runner",
			doc:    "# execute \"/shebang-test/missing-runner\"\n",
			exit:   1,
			stderr: "cannot configure command:",
		},
		{
			name:   "argument validation failure",
			doc:    "# arg \"word\"\n",
			exit:   1,
			stderr: "cannot execute command:",
		},
		{
			name: "exec preserves arguments and environment",
			doc: "# option \"output\" { short \"o\"; }\n" +
				"# flag \"verbose\" { short \"v\"; }\n# arg \"word\"\n\n" +
				"printf '%s|%s|%s|%s' \"$1\" \"$SHEBANG_OL_OUTPUT\" " +
				"\"$SHEBANG_FL_VERBOSE\" \"$SHEBANG_TEST_INHERITED\"\n",
			args:    []string{"-o", "привет", "-v", "word"},
			environ: []string{"SHEBANG_TEST_INHERITED=привет"},
			stdout:  "word|привет|true|привет",
		},
		{
			name: "interpreter exit status preserved",
			doc:  "\nexit 7\n",
			exit: 7,
		},
		{
			name:    "debug does not execute script",
			doc:     "# arg \"word\"\n\nprintf 'SCRIPT RAN'\nexit 7\n",
			args:    []string{"привет"},
			environ: []string{"SHEBANG_DEBUG=1"},
			debug:   true,
		},
		{
			name:       "completion takes precedence over execution",
			doc:        "# arg \"required\"\n\nprintf 'SCRIPT RAN'\n",
			args:       []string{"--unknown"},
			environ:    []string{"SHEBANG_COMPLETION=bash"},
			completion: true,
		},
		{
			name:       "present empty completion uses shell",
			doc:        "\nprintf 'SCRIPT RAN'\n",
			environ:    []string{"SHEBANG_COMPLETION=", "SHELL=/bin/bash"},
			completion: true,
		},
		{
			name:    "unsupported completion shell",
			environ: []string{"SHEBANG_COMPLETION=unsupported"},
			exit:    1,
			stderr:  "cannot generate shell completions: unsupported shell unsupported",
		},
		{
			name:   "help does not execute script",
			doc:    "# description \"привет\"\n\nprintf 'SCRIPT RAN'\n",
			args:   []string{"--help"},
			stdout: "привет",
		},
	} {
		suite.Run(test.name, func() {
			path := filepath.Join(suite.T().TempDir(), "script")

			if !test.missing && !test.noScript {
				doc := "#!shebang.1\n# execute \"" + shellPath + "\"\n" + test.doc
				suite.Require().NoError(os.WriteFile(path, []byte(doc), 0o700))
			}

			args := []string{path}
			if test.noScript {
				args = []string{}
			}

			args = append(args, test.args...)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			executable, err := os.Executable()
			suite.Require().NoError(err)

			processArgs := append([]string{"-test.run=^TestMainProcess$", "--"}, args...)
			cmd := exec.CommandContext(ctx, executable, processArgs...)

			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "SHEBANG_") && !strings.HasPrefix(entry, "SHELL=") {
					cmd.Env = append(cmd.Env, entry)
				}
			}

			cmd.Env = append(cmd.Env, mainProcessMode+"=1")
			cmd.Env = append(cmd.Env, test.environ...)

			var stdout, stderr bytes.Buffer

			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err = cmd.Run()

			suite.Require().NoError(ctx.Err(), "subprocess timed out")

			suite.checkMainResult(
				test,
				err,
				stdout.String(),
				stderr.String(),
				shellPath,
				path,
			)
		})
	}
}

func (suite *MainTestSuite) checkMainResult(
	test mainTestCase,
	err error,
	stdout, stderr string,
	shellPath, path string,
) {
	suite.T().Helper()

	if test.exit == 0 {
		suite.Require().NoError(err, stderr)
	} else {
		var exitError *exec.ExitError

		suite.Require().ErrorAs(err, &exitError)
		suite.Equal(test.exit, exitError.ExitCode(), stderr)
	}

	if test.stdout != "" {
		suite.Contains(stdout, test.stdout)
	}

	if test.stderr != "" {
		suite.Contains(stderr, test.stderr)
	}

	if test.completion {
		suite.Contains(stdout, "bash completion")
	}

	if test.debug {
		suite.Contains(stderr, "Argv: ["+shellPath+" "+path+" привет]")
		suite.NotContains(stderr, "SHEBANG_DEBUG=")
	}

	suite.NotContains(stdout, "SCRIPT RAN")
	suite.NotContains(stderr, "SCRIPT RAN")
}

type mainErrorWriter struct {
	err error
}

func (w mainErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

//nolint:paralleltest // Subprocess helper invokes main, which uses process-global state.
func TestMainProcess(t *testing.T) {
	if os.Getenv(mainProcessMode) != "1" {
		t.Skip("subprocess helper")
	}

	for idx, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"shebang"}, os.Args[idx+1:]...)

			main()
			os.Exit(0)
		}
	}

	os.Exit(97)
}

//nolint:paralleltest // The suite mutates environment variables and os.Args.
func TestMainSuite(t *testing.T) {
	suite.Run(t, &MainTestSuite{})
}
