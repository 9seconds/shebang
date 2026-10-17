package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/glamour/v2"
	"github.com/9seconds/shebang/internal/config/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

const mainProcessMode = "SHEBANG_TEST_MAIN_PROCESS"

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) TestWantsReadme() {
	for _, test := range []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "nil arguments",
			want: true,
		},
		{
			name: "empty arguments",
			args: []string{},
			want: true,
		},
		{
			name: "short help",
			args: []string{"-h"},
			want: true,
		},
		{
			name: "long help",
			args: []string{"--help"},
			want: true,
		},
		{
			name: "script name",
			args: []string{"script"},
		},
		{
			name: "unknown flag",
			args: []string{"--unknown"},
		},
		{
			name: "script help",
			args: []string{
				"script",
				"--help",
			},
		},
		{
			name: "short help with extra argument",
			args: []string{
				"-h",
				"привет",
			},
		},
		{
			name: "long help with extra argument",
			args: []string{
				"--help",
				"привет",
			},
		},
	} {
		suite.Run(test.name, func() {
			// Act
			wanted := wantsReadme(test.args)

			// Assert
			suite.Equal(test.want, wanted)
		})
	}
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
			doc: `#!/bin/sh
echo привет
`,
		},
		{
			name: "embedded configuration",
			doc: `#!shebang.1
# description "привет"
`,
		},
		{
			name:    "missing file",
			missing: true,
			message: "cannot open",
		},
		{
			name: "invalid configuration",
			doc: `#!shebang.1
# unknown
`,
			message: "cannot process node unknown:",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			path := filepath.Join(suite.T().TempDir(), "script")
			if !test.missing {
				suite.Require().NoError(os.WriteFile(path, []byte(test.doc), 0o600))
			}

			// Act
			conf, err := parseConfig(path)

			// Assert
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
			// Arrange
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
			os.Args = []string{
				"shebang",
				input,
			}

			defer func() { os.Args = previous }()

			// Act
			got, err := getScript()

			// Assert
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
			// Arrange
			suite.T().Setenv("SHEBANG_COMPLETION", test.shell)
			suite.T().Setenv("SHELL", test.fallback)

			cmd := &cobra.Command{
				Use: "script",
			}

			var output bytes.Buffer

			cmd.SetOut(&output)

			// Act
			err := runCompletion(cmd)

			// Assert
			suite.Require().NoError(err)
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
			// Arrange
			suite.T().Setenv("SHEBANG_COMPLETION", test.shell)
			suite.T().Setenv("SHELL", test.fallback)

			cmd := &cobra.Command{
				Use: "script",
			}

			var output bytes.Buffer

			cmd.SetOut(&output)

			// Act
			err := runCompletion(cmd)

			// Assert
			suite.Require().ErrorIs(err, errUnsupportedShell)
			suite.Require().EqualError(err, test.want)
			suite.Empty(output.String())
		})
	}
}

func (suite *MainTestSuite) TestCompletionWriterErrors() {
	want := errors.New("write failed")

	for _, shell := range []string{
		"bash",
		"zsh",
		"fish",
		"pwsh",
	} {
		suite.Run(shell, func() {
			// Arrange
			suite.T().Setenv("SHEBANG_COMPLETION", shell)

			cmd := &cobra.Command{
				Use: "script",
			}
			cmd.SetOut(mainErrorWriter{
				err: want,
			})

			// Act
			err := runCompletion(cmd)

			// Assert
			suite.ErrorIs(err, want)
		})
	}
}

func (suite *MainTestSuite) TestRunDebug() {
	// Arrange
	suite.T().Setenv("SHEBANG_TEST_VALUE", "привет=world")
	suite.T().Setenv("OTHER_TEST_VALUE", "do not print")

	var output bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&output)

	// Act
	err := runDebug(cmd, []string{
		"runner",
		"script",
		"привет",
	})

	// Assert
	suite.Require().NoError(err)
	suite.Contains(output.String(), `Argv: [runner script привет]
Environment:
`)
	suite.Contains(output.String(), "SHEBANG_TEST_VALUE=привет=world\n")
	suite.NotContains(output.String(), "OTHER_TEST_VALUE")
}

func (suite *MainTestSuite) TestRunExecveError() {
	// Arrange
	path := filepath.Join(suite.T().TempDir(), "missing")

	// Act
	err := runExecve(&cobra.Command{}, []string{path})

	// Assert
	suite.Require().ErrorIs(err, os.ErrNotExist)
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
	readme     bool
}

//nolint:maintidx // Keep declarative subprocess cases together with one value per line.
func mainTestCases() []mainTestCase {
	return []mainTestCase{
		{
			name:     "README without script",
			noScript: true,
			readme:   true,
		},
		{
			name:     "README with standalone short help flag",
			noScript: true,
			args:     []string{"-h"},
			readme:   true,
		},
		{
			name:     "README with standalone long help flag",
			noScript: true,
			args:     []string{"--help"},
			readme:   true,
		},
		{
			name:     "short help flag with another argument is a script path",
			noScript: true,
			args: []string{
				"-h",
				"extra",
			},
			exit:   1,
			stderr: "cannot detect a script -h:",
		},
		{
			name:     "long help flag with another argument is a script path",
			noScript: true,
			args: []string{
				"--help",
				"extra",
			},
			exit:   1,
			stderr: "cannot detect a script --help:",
		},
		{
			name:     "two help flags do not select README help",
			noScript: true,
			args: []string{
				"-h",
				"--help",
			},
			exit:   1,
			stderr: "cannot detect a script -h:",
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
			doc: `# option "output" { short "o"; }
# flag "verbose" { short "v"; }
# arg "word"

printf '%s|%s|%s|%s' "$1" "$SHEBANG_OL_OUTPUT" \
  "$SHEBANG_FL_VERBOSE" "$SHEBANG_TEST_INHERITED"
`,
			args: []string{
				"-o",
				"привет",
				"-v",
				"word",
			},
			environ: []string{"SHEBANG_TEST_INHERITED=привет"},
			stdout:  "word|привет|true|привет",
		},
		{
			name: "hyphenated CLI names export shell-readable variables",
			doc: `# option "archive-name" { short "a"; }
# flag "local-only"
# arg "source-dir"

printf '%s|%s|%s|%s' "$1" "$SHEBANG_OL_ARCHIVE_NAME" \
  "$SHEBANG_OS_A" "$SHEBANG_FL_LOCAL_ONLY"
`,
			args: []string{
				"--archive-name",
				"привет",
				"--local-only",
				"source",
			},
			stdout: "source|привет|привет|true",
		},
		{
			name: "integer validation preserves original values",
			doc: `# option "limit" {
#   value "int" { min 1; max 20; }
# }
# arg "first" { value "int"; }
# vararg "items" { value "int"; min-count 1; }
# arg "last" { value "int"; }

printf '%s|%s|%s|%s' "$SHEBANG_OL_LIMIT" "$1" "$2" "$3"
`,
			args: []string{
				"--limit",
				"+0012",
				"--",
				"0002",
				"-3",
				"04",
			},
			stdout: "+0012|0002|-3|04",
		},
		{
			name: "invalid integer option prevents execution",
			doc: `# option "limit" { value "int" { min 1; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"--limit",
				"0",
			},
			exit:   1,
			stderr: "number must be at least 1, got 0",
		},
		{
			name: "float validation preserves original values",
			doc: `# option "ratio" { value "float" { min 0.0; max 1.0; }; }
# arg "offset" { value "float"; }
# vararg "samples" { value "float"; min-count 1; }
# arg "scale" { value "float"; }

printf '%s|%s|%s|%s' "$SHEBANG_OL_RATIO" "$1" "$2" "$3"
`,
			args: []string{
				"--ratio",
				"+05e-1",
				"--",
				"-1.25",
				".5",
				"2.00",
			},
			stdout: "+05e-1|-1.25|.5|2.00",
		},
		{
			name: "NaN cannot bypass float bounds",
			doc: `# option "ratio" { value "float" { min 0.0; max 1.0; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"--ratio",
				"NaN",
			},
			exit:   1,
			stderr: "float must be finite",
		},
		{
			name: "IP validation preserves original spelling",
			doc: `# option "address" { value "ip" { type "v6-only"; }; }
# arg "peer" { value "ip" { private #true; }; }

printf '%s|%s' "$SHEBANG_OL_ADDRESS" "$1"
`,
			args: []string{
				"--address",
				"2001:DB8:0:0:0:0:0:1",
				"10.1.2.3",
			},
			stdout: "2001:DB8:0:0:0:0:0:1|10.1.2.3",
		},
		{
			name: "IP subnet rejection prevents execution",
			doc: `# arg "peer" { value "ip" { subnets "192.0.2.0/24"; }; }

printf 'SCRIPT RAN'
`,
			args:   []string{"198.51.100.1"},
			exit:   1,
			stderr: "does not belong to any configured subnet",
		},
		{
			name: "port validation preserves original spelling",
			doc: `# option "port" { value "port" { registered #true; }; }
# arg "peer-port" { value "port"; }

printf '%s|%s' "$SHEBANG_OL_PORT" "$1"
`,
			args: []string{
				"--port",
				"08080",
				"00080",
			},
			stdout: "08080|00080",
		},
		{
			name: "port range rejection prevents execution",
			doc: `# option "port" { value "port" { well-known #false; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"--port",
				"80",
			},
			exit:   1,
			stderr: "port 80 is well-known",
		},
		{
			name: "enum option and positional values reach the script",
			doc: `# option "mode" { value "enum" { choices "dev" "prod"; }; }
# arg "greeting" { value "enum" { choices "привет"; }; }

printf '%s|%s' "$SHEBANG_OL_MODE" "$1"
`,
			args: []string{
				"--mode",
				"prod",
				"привет",
			},
			stdout: "prod|привет",
		},
		{
			name: "unknown enum choice prevents execution",
			doc: `# option "mode" { value "enum" { choices "dev" "prod"; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"--mode",
				"preview",
			},
			exit:   1,
			stderr: "unknown choice",
		},
		{
			name: "enum choices complete through Cobra protocol",
			doc: `# option "mode" { value "enum" { choices "production" "preview" "dev"; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"--mode",
				"pr",
			},
			stdout: `preview
production
:4
`,
		},
		{
			name: "interpreter exit status preserved",
			doc: `
exit 7
`,
			exit: 7,
		},
		{
			name: "Git syntax validation preserves option and positional values",
			doc: `# option "revision" { short "r"; value "git"; }
# arg "branch" { value "git"; }

printf '%s|%s|%s' "$SHEBANG_OL_REVISION" "$SHEBANG_OS_R" "$1"
`,
			args: []string{
				"--revision",
				"refs/tags/v1.2",
				"feature/привет",
			},
			stdout: "refs/tags/v1.2|refs/tags/v1.2|feature/привет",
		},
		{
			name: "invalid Git reference syntax prevents execution",
			doc: `# arg "revision" { value "git"; }

printf 'SCRIPT RAN'
`,
			args:   []string{"main~1"},
			exit:   1,
			stderr: "invalid reference name",
		},
		{
			name: "unknown Git category prevents configuration",
			doc: `# arg "revision" { value "git" { type "any" "typo"; }; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "unknown git ref type: typo",
		},
		{
			name: "subnet preserves original option and positional input",
			doc: `# option "network" {
#   short "n"
#   value "subnet" {
#     type "v4"
#     overlaps "10.0.0.0/8" "192.168.0.0/16"
#     not-overlaps "10.1.0.0/16"
#   }
# }
# arg "peer" { value "subnet"; }

printf '%s|%s|%s' "$SHEBANG_OL_NETWORK" "$SHEBANG_OS_N" "$1"
`,
			args: []string{
				"--network",
				"192.168.1.19/24",
				"2001:DB8::1/32",
			},
			stdout: "192.168.1.19/24|192.168.1.19/24|2001:DB8::1/32",
		},
		{
			name: "subnet exclusion prevents execution",
			doc: `# arg "network" { value "subnet" { not-overlaps "10.0.0.0/8"; }; }

printf 'SCRIPT RAN'
`,
			args:   []string{"10.1.2.3/24"},
			exit:   1,
			stderr: "must not overlap",
		},
		{
			name: "subnet overlap rejection prevents execution",
			doc: `# arg "network" { value "subnet" { overlaps "10.0.0.0/8"; }; }

printf 'SCRIPT RAN'
`,
			args:   []string{"192.0.2.0/24"},
			exit:   1,
			stderr: "must overlap with at least one subnet",
		},
		{
			name: "subnet empty overlaps prevents configuration",
			doc: `# arg "network" { value "subnet" { overlaps; }; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "no elements were defined",
		},
		{
			name: "subnet option completion masks and filters family",
			doc: `# option "network" {
#   value "subnet" { type "v4"; overlaps "192.0.2.19/24" "2001:db8::/32"; }
# }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"--network",
				"",
			},
			stdout: "192.0.2.0/\n:2\n",
		},
		{
			name: "subnet exclusions disable partial positional suggestions",
			doc: `# arg "network" {
#   value "subnet" { overlaps "10.0.0.0/8"; not-overlaps "10.1.0.0/16"; }
# }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"10.",
			},
			stdout: ":4\n",
		},
		{
			name: "URL preserves option and positional spelling with combined constraints",
			doc: `# option "endpoint" {
#   short "e"
#   value "url" {
#     scheme "https"
#     domain "EXAMPLE.COM"
#     user "alice"
#     password #false
#     path-glob "/api/**"
#     query "mode" "safe"
#     query-re "id" "^[0-9]+$"
#     fragment-re "^done$"
#   }
# }
# arg "reference" { value "url"; }

printf '%s|%s|%s' "$SHEBANG_OL_ENDPOINT" "$SHEBANG_OS_E" "$1"
`,
			args: []string{
				"--endpoint",
				"HTTPS://%61lice@EXAMPLE.COM:443/api/a%2Fb?mode=safe&id=01&id=02#done",
				"../привет",
			},
			stdout: "HTTPS://%61lice@EXAMPLE.COM:443/api/a%2Fb?mode=safe&id=01&id=02#done|" +
				"HTTPS://%61lice@EXAMPLE.COM:443/api/a%2Fb?mode=safe&id=01&id=02#done|../привет",
		},
		{
			name: "URL rejects mismatching repeated query values before execution",
			doc: `# arg "endpoint" { value "url" { query "mode" "safe"; }; }

printf 'SCRIPT RAN'
`,
			args:   []string{"https://example.com?mode=safe&mode=unsafe"},
			exit:   1,
			stderr: `URL query mode must equal "safe", got "unsafe"`,
		},
		{
			name: "URL malformed escape prevents execution",
			doc: `# arg "endpoint" { value "url"; }

printf 'SCRIPT RAN'
`,
			args:   []string{"/%zz"},
			exit:   1,
			stderr: "invalid URL escape",
		},
		{
			name: "URL duplicate query keys prevent configuration",
			doc: `# arg "endpoint" { value "url" { query "x" "1" "x" "2"; }; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "duplicate query parameter",
		},
		{
			name: "URL option completion preserves original text",
			doc: `# option "endpoint" { value "url" { scheme "https"; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"--endpoint",
				"HTTPS://EXAMPLE.COM/a%2Fb",
			},
			stdout: "HTTPS://EXAMPLE.COM/a%2Fb\n:4\n",
		},
		{
			name: "URL rejected positional completion has no suggestions",
			doc: `# arg "endpoint" { value "url" { scheme "https"; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"http://example.com",
			},
			stdout: ":4\n",
		},
		{
			name: "semver preserves original option and positional values",
			doc: `# option "release" {
#   short "r"
#   value "semver" { constraint ">= 1.2.0 < 2.0.0"; }
# }
# arg "baseline" { value "semver"; }

printf '%s|%s|%s' "$SHEBANG_OL_RELEASE" "$SHEBANG_OS_R" "$1"
`,
			args: []string{
				"--release",
				"v01.02.003+build.1",
				"v1.2",
			},
			stdout: "v01.02.003+build.1|v01.02.003+build.1|v1.2",
		},
		{
			name: "semver constraint prevents execution",
			doc: `# arg "release" { value "semver" { constraint "< 2.0.0"; }; }

printf 'SCRIPT RAN'
`,
			args:   []string{"2.0.0"},
			exit:   1,
			stderr: `"2.0.0" is greater than or equal to "2.0.0"`,
		},
		{
			name: "invalid semver prevents execution",
			doc: `# arg "release" { value "semver"; }

printf 'SCRIPT RAN'
`,
			args:   []string{"привет"},
			exit:   1,
			stderr: "invalid semantic version",
		},
		{
			name: "invalid semver constraint prevents configuration",
			doc: `# arg "release" { value "semver" { constraint "garbage"; }; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "incorrect constraint:",
		},
		{
			name: "semver completion preserves shorthand",
			doc: `# option "release" { value "semver" { constraint "^1.0.0"; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"--release",
				"v1.2",
			},
			stdout: "v1.2\n:4\n",
		},
		{
			name: "semver constrained completion has no invalid suggestions",
			doc: `# arg "release" { value "semver" { constraint "^1.0.0"; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"2.0.0",
			},
			stdout: ":4\n",
		},
		{
			name: "JSON option and scalar arguments preserve original text",
			doc: `# option "payload" { short "p"; value "json"; }
# arg "first" { value "json"; }
# vararg "items" { value "json"; min-count 1; }

printf '%s|%s|%s|%s|%s' "$SHEBANG_OL_PAYLOAD" "$SHEBANG_OS_P" "$1" "$2" "$3"
`,
			args: []string{
				"--payload",
				` { "greeting": "привет", "nested": [null, {"x":1}] } `,
				"null",
				"1e1000",
				`"привет"`,
			},
			stdout: ` { "greeting": "привет", "nested": [null, {"x":1}] } |` +
				` { "greeting": "привет", "nested": [null, {"x":1}] } |null|1e1000|"привет"`,
		},
		{
			name: "invalid JSON option prevents execution with syntax error",
			doc: `# option "payload" { value "json"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"--payload",
				`{"a":}`,
			},
			exit:   1,
			stderr: "looking for beginning of value",
		},
		{
			name: "multiple JSON positional values in one argument prevent execution",
			doc: `# arg "payload" { value "json"; }

printf 'SCRIPT RAN'
`,
			args:   []string{"1 2"},
			exit:   1,
			stderr: "after top-level value",
		},
		{
			name: "JSON option completion preserves original text",
			doc: `# option "payload" { value "json"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"--payload",
				`{"greeting":"привет"}`,
			},
			stdout: "{\"greeting\":\"привет\"}\n:4\n",
		},
		{
			name: "incomplete JSON positional completion has no suggestions",
			doc: `# arg "payload" { value "json"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"{",
			},
			stdout: ":4\n",
		},
		{
			name: "JSON subvalidator prevents configuration",
			doc: `# arg "payload" { value "json" { schema "object"; }; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "does not support subvalidators",
		},
		{
			name: "email preserves option and positional spelling",
			doc: `# option "recipient" {
#   short "r"
#   value "email" { domain "EXAMPLE.COM"; user-re "^Alice"; icann #true; }
# }
# arg "sender" { value "email" { domain-re "^example[.]com$"; }; }

printf '%s|%s|%s' "$SHEBANG_OL_RECIPIENT" "$SHEBANG_OS_R" "$1"
`,
			args: []string{
				"--recipient",
				"Alice+tag@EXAMPLE.COM",
				"sender@example.com",
			},
			stdout: "Alice+tag@EXAMPLE.COM|Alice+tag@EXAMPLE.COM|sender@example.com",
		},
		{
			name: "email domain rejection prevents execution",
			doc: `# option "recipient" { value "email" { domain "example.com"; }; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"--recipient",
				"alice@example.org",
			},
			exit:   1,
			stderr: "unsupported domain",
		},
		{
			name: "empty email prevents execution without panic",
			doc: `# arg "recipient" { value "email"; }

printf 'SCRIPT RAN'
`,
			args:   []string{""},
			exit:   1,
			stderr: "email address cannot be empty",
		},
		{
			name: "ICANN required rejection prevents execution",
			doc: `# arg "recipient" { value "email" { icann #true; }; }

printf 'SCRIPT RAN'
`,
			args:   []string{"alice@example.invalid"},
			exit:   1,
			stderr: "email domain must have an ICANN-managed public suffix",
		},
		{
			name: "ICANN excluded rejection prevents execution",
			doc: `# arg "recipient" { value "email" { icann #false; }; }

printf 'SCRIPT RAN'
`,
			args:   []string{"alice@example.com"},
			exit:   1,
			stderr: "email domain must not have an ICANN-managed public suffix",
		},
		{
			name: "email option completion preserves input",
			doc: `# option "recipient" { value "email"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"--recipient",
				"Alice@EXAMPLE.COM",
			},
			stdout: "Alice@EXAMPLE.COM\n:4\n",
		},
		{
			name: "empty email completion has no suggestions or panic",
			doc: `# arg "recipient" { value "email"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"",
			},
			stdout: ":4\n",
		},
		{
			name: "invalid email regex prevents configuration",
			doc: `# arg "recipient" { value "email" { user-re "["; }; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "cannot add subvalidator user-re",
		},
		{
			name: "UUID option and positional preserve original spelling",
			doc: `# option "request-id" { short "r"; value "uuid"; }
# arg "resource-id" { value "uuid"; }
# vararg "related-id" { value "uuid"; min-count 1; }

printf '%s|%s|%s|%s' "$SHEBANG_OL_REQUEST_ID" "$SHEBANG_OS_R" "$1" "$2"
`,
			args: []string{
				"--request-id",
				"F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6",
				"{550e8400-e29b-41d4-a716-446655440000}",
				"f81d4fae7dec11d0a76500a0c91e6bf6",
			},
			stdout: "F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6|" +
				"F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6|" +
				"{550e8400-e29b-41d4-a716-446655440000}|f81d4fae7dec11d0a76500a0c91e6bf6",
		},
		{
			name: "MAC address preserves option and positional spelling",
			doc: `# option "source" { short "s"; value "mac-addr"; }
# arg "destination" { value "mac-addr"; }
# vararg "peer" { value "mac-addr"; min-count 1; }

printf '%s|%s|%s|%s' "$SHEBANG_OL_SOURCE" "$SHEBANG_OS_S" "$1" "$2"
`,
			args: []string{
				"--source",
				"AA-BB-CC-DD-EE-FF",
				"0000.5e00.5301",
				"02005e1000000001",
			},
			stdout: "AA-BB-CC-DD-EE-FF|AA-BB-CC-DD-EE-FF|0000.5e00.5301|02005e1000000001",
		},
		{
			name: "invalid MAC option prevents execution",
			doc: `# option "source" { value "mac-addr"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"--source",
				"привет",
			},
			exit:   1,
			stderr: "invalid MAC address",
		},
		{
			name: "invalid MAC positional prevents execution",
			doc: `# arg "destination" { value "mac-addr"; }

printf 'SCRIPT RAN'
`,
			args:   []string{"00:00:5e"},
			exit:   1,
			stderr: "invalid MAC address",
		},
		{
			name: "MAC property prevents configuration",
			doc: `# arg "destination" { value "mac-addr" format="colon"; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "does not support properties",
		},
		{
			name: "MAC subvalidator prevents configuration",
			doc: `# arg "destination" { value "mac-addr" { length 6; }; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "does not support subvalidators",
		},
		{
			name: "MAC option completion preserves uppercase hyphens",
			doc: `# option "source" { value "mac-addr"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"--source",
				"AA-BB-CC-DD-EE-FF",
			},
			stdout: "AA-BB-CC-DD-EE-FF\n:4\n",
		},
		{
			name: "partial MAC positional completion has no suggestions",
			doc: `# arg "destination" { value "mac-addr"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"00:00:5e",
			},
			stdout: ":4\n",
		},
		{
			name: "invalid UUID option prevents execution",
			doc: `# option "request-id" { value "uuid"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"--request-id",
				"привет",
			},
			exit:   1,
			stderr: "invalid uuid",
		},
		{
			name: "invalid UUID positional prevents execution",
			doc: `# arg "resource-id" { value "uuid"; }

printf 'SCRIPT RAN'
`,
			args:   []string{"f81d4fae"},
			exit:   1,
			stderr: "invalid uuid",
		},
		{
			name: "UUID property prevents configuration",
			doc: `# arg "resource-id" { value "uuid" format="urn"; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "does not support properties",
		},
		{
			name: "UUID subvalidator prevents configuration",
			doc: `# arg "resource-id" { value "uuid" { version 4; }; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "does not support subvalidators",
		},
		{
			name: "UUID option completion preserves uppercase input",
			doc: `# option "request-id" { value "uuid"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"--request-id",
				"F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6",
			},
			stdout: "F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6\n:4\n",
		},
		{
			name: "partial UUID positional completion has no suggestions",
			doc: `# arg "resource-id" { value "uuid"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"f81d4fae",
			},
			stdout: ":4\n",
		},
		{
			name: "base64 preserves encoded option and positional values",
			doc: `# option "payload" {
#   short "p"
#   value "base64" format="raw-url" { min-bytes 2; max-bytes 2; }
# }
# arg "greeting" { value "base64" { min-bytes 12; max-bytes 12; }; }

printf '%s|%s|%s' "$SHEBANG_OL_PAYLOAD" "$SHEBANG_OS_P" "$1"
`,
			args: []string{
				"--payload",
				"-_8",
				"0L/RgNC40LLQtdGC",
			},
			stdout: "-_8|-_8|0L/RgNC40LLQtdGC",
		},
		{
			name: "base64 decoded byte limit prevents execution",
			doc: `# arg "payload" { value "base64" { max-bytes 1; }; }

printf 'SCRIPT RAN'
`,
			args:   []string{"Zm8="},
			exit:   1,
			stderr: "length must be at most 1 bytes",
		},
		{
			name: "malformed base64 prevents execution",
			doc: `# arg "payload" { value "base64"; }

printf 'SCRIPT RAN'
`,
			args:   []string{"Zg"},
			exit:   1,
			stderr: "illegal base64 data",
		},
		{
			name: "base64 completion preserves input through Cobra",
			doc: `# option "payload" { value "base64" format="raw-url"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"--payload",
				"Zm8",
			},
			stdout: "Zm8\n:4\n",
		},
		{
			name: "invalid base64 format prevents execution",
			doc: `# arg "payload" { value "base64" format="STD"; }

printf 'SCRIPT RAN'
`,
			exit:   1,
			stderr: "unknown format STD",
		},
		{
			name: "base64 invalid completion has no suggestions",
			doc: `# arg "payload" { value "base64"; }

printf 'SCRIPT RAN'
`,
			args: []string{
				"__complete",
				"Zg",
			},
			stdout: ":4\n",
		},
		{
			name: "filepath option and positional preserve original paths",
			doc: `# option "input" { value "filepath" { glob "*.txt"; }; }
# arg "directory" { value "filepath" { type "directory"; }; }

printf '%s|%s' "$SHEBANG_OL_INPUT" "$1"
`,
			args: []string{
				"--input",
				"./привет.txt",
				".",
			},
			stdout: "./привет.txt|.",
		},
		{
			name: "filepath type rejection prevents execution",
			doc: `# arg "input" { value "filepath" { type "regular"; }; }

printf 'SCRIPT RAN'
`,
			args:   []string{"."},
			exit:   1,
			stderr: "path has no suitable type",
		},
		{
			name: "filepath empty input prevents execution",
			doc: `# arg "input" { value "filepath"; }

printf 'SCRIPT RAN'
`,
			args:   []string{""},
			exit:   1,
			stderr: "no path is defined",
		},
		{
			name: "debug does not execute script",
			doc: `# arg "word"

printf 'SCRIPT RAN'
exit 7
`,
			args:    []string{"привет"},
			environ: []string{"SHEBANG_DEBUG=1"},
			debug:   true,
		},
		{
			name: "completion takes precedence over execution",
			doc: `# arg "required"

printf 'SCRIPT RAN'
`,
			args:       []string{"--unknown"},
			environ:    []string{"SHEBANG_COMPLETION=bash"},
			completion: true,
		},
		{
			name: "present empty completion uses shell",
			doc: `
printf 'SCRIPT RAN'
`,
			environ: []string{
				"SHEBANG_COMPLETION=",
				"SHELL=/bin/bash",
			},
			completion: true,
		},
		{
			name:    "unsupported completion shell",
			environ: []string{"SHEBANG_COMPLETION=unsupported"},
			exit:    1,
			stderr:  "cannot generate shell completions: unsupported shell unsupported",
		},
		{
			name: "help does not execute script",
			doc: `# description "привет"

printf 'SCRIPT RAN'
`,
			args:   []string{"--help"},
			stdout: "привет",
		},
	}
}

func (suite *MainTestSuite) TestMain() {
	shellPath, err := exec.LookPath("sh")
	suite.Require().NoError(err)

	for _, test := range mainTestCases() {
		suite.Run(test.name, func() {
			// Arrange
			path := filepath.Join(suite.T().TempDir(), "script")

			if !test.missing && !test.noScript {
				doc := fmt.Sprintf(`#!shebang.1
# execute %q
%s`, shellPath, test.doc)
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

			processArgs := append([]string{
				"-test.run=^TestMainProcess$",
				"--",
			}, args...)
			cmd := exec.CommandContext(ctx, executable, processArgs...)

			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "SHEBANG_") &&
					!strings.HasPrefix(entry, "SHELL=") && !strings.HasPrefix(entry, "GLAMOUR_STYLE=") {
					cmd.Env = append(cmd.Env, entry)
				}
			}

			cmd.Env = append(cmd.Env, mainProcessMode+"=1")
			cmd.Env = append(cmd.Env, "GLAMOUR_STYLE=notty")
			cmd.Env = append(cmd.Env, test.environ...)

			var stdout, stderr bytes.Buffer

			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			// Act
			err = cmd.Run()

			// Assert
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

	if test.readme {
		rendered, renderError := glamour.Render(readmeContent, "notty")
		suite.Require().NoError(renderError)
		suite.Equal(rendered+"\n", stdout)
		suite.Empty(stderr)
	}

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
