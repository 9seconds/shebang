package main

import (
	"bytes"
	"context"
	"errors"
	"io"
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
				suite.ErrorContains(err, test.message)
				suite.Nil(conf)

				if test.missing {
					suite.ErrorIs(err, os.ErrNotExist)
				}
			} else {
				suite.Require().NoError(err)
				suite.Require().IsType(&v1.Config{}, conf)

				parsed := conf.(*v1.Config)
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
				suite.Error(err)
				suite.Empty(got)
			} else {
				suite.NoError(err)
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
			name:   "powershell",
			shell:  "powershell",
			marker: "Register-ArgumentCompleter",
		},
		{
			name:   "power alias",
			shell:  "power",
			marker: "Register-ArgumentCompleter",
		},
		{
			name:     "auto uses shell environment",
			shell:    "auto",
			fallback: "/bin/zsh",
			marker:   "#compdef script",
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

func (suite *MainTestSuite) TestCompletionWriterErrors() {
	want := errors.New("write failed")
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
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
	fp, err := os.CreateTemp(suite.T().TempDir(), "stdout")
	suite.Require().NoError(err)
	previous := os.Stdout
	os.Stdout = fp
	defer func() {
		os.Stdout = previous
		_ = fp.Close()
	}()
	suite.NoError(runDebug([]string{"runner", "script", "привет"}))
	_, err = fp.Seek(0, io.SeekStart)
	suite.Require().NoError(err)
	output, err := io.ReadAll(fp)
	suite.Require().NoError(err)
	suite.Contains(string(output), "Argv: [runner script привет]\nEnvironment:\n")
	suite.Contains(string(output), "SHEBANG_TEST_VALUE=привет=world\n")
	suite.NotContains(string(output), "OTHER_TEST_VALUE")
}

func (suite *MainTestSuite) TestRunExecveError() {
	path := filepath.Join(suite.T().TempDir(), "missing")
	suite.ErrorIs(runExecve([]string{path}), os.ErrNotExist)
}

func (suite *MainTestSuite) TestMain() {
	sh, err := exec.LookPath("sh")
	suite.Require().NoError(err)
	for _, test := range []struct {
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
	}{
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
			name:    "exec preserves arguments and environment",
			doc:     "# option \"output\" { short \"o\"; }\n# flag \"verbose\" { short \"v\"; }\n# arg \"word\"\n\nprintf '%s|%s|%s|%s' \"$1\" \"$SHEBANG_OL_OUTPUT\" \"$SHEBANG_FL_VERBOSE\" \"$SHEBANG_TEST_INHERITED\"\n",
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
			stderr:  "unsupported shell unsupported",
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
				doc := "#!shebang.1\n# execute \"" + sh + "\"\n" + test.doc
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
			cmd := exec.CommandContext(ctx, executable, append([]string{"-test.run=^TestMainProcess$", "--"}, args...)...)
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
			if test.exit == 0 {
				suite.Require().NoError(err, stderr.String())
			} else {
				var exitError *exec.ExitError
				suite.Require().ErrorAs(err, &exitError)
				suite.Equal(test.exit, exitError.ExitCode(), stderr.String())
			}
			if test.stdout != "" {
				suite.Contains(stdout.String(), test.stdout)
			}
			if test.stderr != "" {
				suite.Contains(stderr.String(), test.stderr)
			}
			if test.completion {
				suite.Contains(stdout.String(), "bash completion")
			}
			if test.debug {
				suite.Contains(stdout.String(), "Argv: ["+sh+" "+path+" привет]")
				suite.NotContains(stdout.String(), "SHEBANG_DEBUG=")
			}
			suite.NotContains(stdout.String(), "SCRIPT RAN")
		})
	}
}

type mainErrorWriter struct {
	err error
}

func (w mainErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

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

func TestMainSuite(t *testing.T) {
	suite.Run(t, &MainTestSuite{})
}
