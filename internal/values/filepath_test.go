package values_test

import (
	"io/fs"
	"math"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type FilepathTestSuite struct {
	suite.Suite
}

func (suite *FilepathTestSuite) TestDirectoryNavigation() {
	// Arrange
	root := suite.T().TempDir()
	directory := filepath.Join(root, "привет")
	linkedDirectory := filepath.Join(root, "ссылка")
	danglingLink := filepath.Join(root, "dangling")

	suite.Require().NoError(os.Mkdir(directory, 0o755))
	suite.Require().NoError(os.Symlink(directory, linkedDirectory))
	suite.Require().NoError(os.Symlink(filepath.Join(root, "missing"), danglingLink))
	suite.Require().NoError(os.WriteFile(filepath.Join(directory, "report.txt"), nil, 0o600))
	suite.Require().NoError(os.WriteFile(filepath.Join(directory, "report.json"), nil, 0o600))

	value, err := values.NewFilepath(nil, map[string][]any{
		"type": {"regular"},
	})
	suite.Require().NoError(err)

	suite.Require().Error(value.Validate(directory))
	suite.Require().Error(value.Validate(linkedDirectory))

	for _, test := range []struct {
		name      string
		input     string
		want      []string
		directive cobra.ShellCompDirective
	}{
		{
			name:  "directory and symlinked directory despite file constraints",
			input: root + string(os.PathSeparator),
			want: []string{
				directory + string(os.PathSeparator),
				linkedDirectory + string(os.PathSeparator),
			},
			directive: cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace,
		},
		{
			name:      "unicode directory prefix",
			input:     filepath.Join(root, "при"),
			want:      []string{directory + string(os.PathSeparator)},
			directive: cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace,
		},
		{
			name:      "symlinked directory prefix",
			input:     filepath.Join(root, "ссы"),
			want:      []string{linkedDirectory + string(os.PathSeparator)},
			directive: cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace,
		},
		{
			name:  "files inside directory match prefix",
			input: filepath.Join(directory, "rep"),
			want: []string{
				filepath.Join(directory, "report.json"),
				filepath.Join(directory, "report.txt"),
			},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
	} {
		suite.Run(test.name, func() {
			// Act
			paths, directive := value.Complete(test.input)

			// Assert
			suite.Equal(test.want, paths)
			suite.Equal(test.directive, directive)
		})
	}
}

func (suite *FilepathTestSuite) TestTypes() {
	// Arrange
	root := suite.T().TempDir()
	file := filepath.Join(root, "привет.txt")
	missing := filepath.Join(root, "missing")
	link := filepath.Join(root, "link")
	dangling := filepath.Join(root, "dangling")
	pipe := filepath.Join(root, "pipe")
	socket := filepath.Join(root, "socket")

	suite.Require().NoError(os.WriteFile(file, nil, 0o600))
	suite.Require().NoError(os.Symlink(file, link))
	suite.Require().NoError(os.Symlink(missing, dangling))
	suite.Require().NoError(syscall.Mkfifo(pipe, 0o600))

	listener, err := net.Listen("unix", socket)
	suite.Require().NoError(err)
	suite.T().Cleanup(func() { suite.NoError(listener.Close()) })

	for _, test := range []struct {
		name     string
		accepted string
		rejected string
	}{
		{
			name:     "exists",
			accepted: link,
			rejected: missing,
		},
		{
			name:     "absent",
			accepted: missing,
			rejected: file,
		},
		{
			name:     "regular",
			accepted: link,
			rejected: root,
		},
		{
			name:     "directory",
			accepted: root,
			rejected: file,
		},
		{
			name:     "symlink",
			accepted: dangling,
			rejected: file,
		},
		{
			name:     "pipe",
			accepted: pipe,
			rejected: file,
		},
		{
			name:     "socket",
			accepted: socket,
			rejected: file,
		},
		{
			name:     "device",
			accepted: "/dev/null",
			rejected: file,
		},
		{
			name:     "char-device",
			accepted: "/dev/null",
			rejected: file,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.New("filepath", nil, map[string][]any{
				"type": {test.name},
			})
			suite.Require().NoError(err)

			// Act
			acceptedError := value.Validate(test.accepted)
			rejectedError := value.Validate(test.rejected)

			// Assert
			suite.Require().NoError(acceptedError)
			suite.Require().ErrorIs(rejectedError, values.ErrFilepathNoSuitableType)
		})
	}

	// Act
	value, err := values.NewFilepath(nil, map[string][]any{
		"type": {
			"regular",
			"directory",
			"regular",
		},
	})

	// Assert
	suite.Require().NoError(err)
	suite.Equal("filepath(type:directory,regular)", value.String())
	suite.Require().NoError(value.Validate(file))
	suite.Require().NoError(value.Validate(root))
}

func (suite *FilepathTestSuite) TestPathsAndConstraints() {
	// Arrange
	root := suite.T().TempDir()
	file := filepath.Join(root, " привет.txt ")
	suite.Require().NoError(os.WriteFile(file, nil, 0o600))
	suite.Require().NoError(os.Chmod(file, 0o600))

	for _, test := range []struct {
		name          string
		subvalidators map[string][]any
		input         string
		want          error
	}{
		{
			name: "empty path",
			want: values.ErrFilepathNotSet,
		},
		{
			name:  "unconstrained missing path",
			input: "привет",
		},
		{
			name:  "space is a filename",
			input: " ",
		},
		{
			name: "preserves filename spaces",
			subvalidators: map[string][]any{
				"type": {"exists"},
			},
			input: file,
		},
		{
			name: "trimmed filename is different",
			subvalidators: map[string][]any{
				"type": {"exists"},
			},
			input: filepath.Join(root, "привет.txt"),
			want:  values.ErrFilepathNoSuitableType,
		},
		{
			name: "glob accepts directory itself",
			subvalidators: map[string][]any{
				"glob": {filepath.Join(root, "**")},
			},
			input: root,
		},
		{
			name: "glob accepts new file in directory",
			subvalidators: map[string][]any{
				"glob": {filepath.Join(root, "**")},
			},
			input: filepath.Join(root, "new.txt"),
		},
		{
			name: "glob excludes sibling directory",
			subvalidators: map[string][]any{
				"glob": {filepath.Join(root, "**")},
			},
			input: root + "-other/new.txt",
			want:  values.ErrFilepathGlobMismatch,
		},
		{
			name: "glob excludes parent directory",
			subvalidators: map[string][]any{
				"glob": {filepath.Join(root, "**")},
			},
			input: filepath.Join(root, "..", "outside"),
			want:  values.ErrFilepathGlobMismatch,
		},
		{
			name: "matching permissions",
			subvalidators: map[string][]any{
				"permissions": {int64(0o600)},
			},
			input: file,
		},
		{
			name: "permissions must match exactly",
			subvalidators: map[string][]any{
				"permissions": {int64(0o644)},
			},
			input: file,
			want:  values.ErrFilepathIncorrectPermissions,
		},
		{
			name: "missing permissions target",
			subvalidators: map[string][]any{
				"permissions": {int64(0o644)},
			},
			input: filepath.Join(root, "missing"),
			want:  fs.ErrNotExist,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewFilepath(nil, test.subvalidators)
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)

			// Assert
			suite.Require().ErrorIs(err, test.want)
		})
	}
}

func (suite *FilepathTestSuite) TestGlobValidation() {
	for _, test := range []struct {
		name    string
		pattern string
		input   string
		valid   bool
	}{
		{
			name:    "extension alternatives",
			pattern: "*.{txt,json}",
			input:   "привет.txt",
			valid:   true,
		},
		{
			name:    "second alternative",
			pattern: "*.{txt,json}",
			input:   "report.json",
			valid:   true,
		},
		{
			name:    "case sensitive",
			pattern: "*.txt",
			input:   "report.TXT",
		},
		{
			name:    "final extension only",
			pattern: "*.txt",
			input:   "report.txt.gz",
		},
		{
			name:    "dotfile",
			pattern: ".env",
			input:   ".env",
			valid:   true,
		},
		{
			name:    "literal trailing dot",
			pattern: "*.",
			input:   "report.",
			valid:   true,
		},
		{
			name:    "extensionless file",
			pattern: "*.",
			input:   "report",
		},
		{
			name:    "nonmatching literal",
			pattern: "other",
			input:   "report.txt",
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			value, err := values.NewFilepath(nil, map[string][]any{
				"glob": {test.pattern},
			})
			suite.Require().NoError(err)

			// Act
			err = value.Validate(test.input)

			// Assert
			if test.valid {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, values.ErrFilepathGlobMismatch)
			}
		})
	}
}

func (suite *FilepathTestSuite) TestConfigurationErrors() {
	for _, test := range []struct {
		name      string
		arguments []any
		want      error
	}{
		{
			name: "unknown",
			want: values.ErrUnknownSubvalidator,
		},
		{
			name: "glob",
			want: utils.ErrEmpty,
		},
		{
			name: "root",
			want: values.ErrUnknownSubvalidator,
		},
		{
			name: "file-extensions",
			want: values.ErrUnknownSubvalidator,
		},
		{
			name:      "glob",
			arguments: []any{"["},
			want:      values.ErrFilepathInvalidGlob,
		},
		{
			name:      "permissions",
			arguments: []any{int64(-1)},
			want:      values.ErrFilepathIncorrectPermissions,
		},
		{
			name:      "permissions",
			arguments: []any{int64(0o1000)},
			want:      values.ErrFilepathIncorrectPermissions,
		},
		{
			name:      "owner-id",
			arguments: []any{int64(-1)},
			want:      values.ErrFilepathIncorrectUID,
		},
		{
			name:      "owner-id",
			arguments: []any{int64(math.MaxUint32) + 1},
			want:      values.ErrFilepathIncorrectUID,
		},
		{
			name:      "group-id",
			arguments: []any{int64(-1)},
			want:      values.ErrFilepathIncorrectGID,
		},
		{
			name:      "group-id",
			arguments: []any{int64(math.MaxUint32) + 1},
			want:      values.ErrFilepathIncorrectGID,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			properties := map[string][]any{
				test.name: test.arguments,
			}

			// Act
			value, err := values.NewFilepath(nil, properties)

			// Assert
			suite.Nil(value)
			suite.Require().ErrorIs(err, test.want)
		})
	}

	for _, name := range []string{
		"glob",
		"type",
		"permissions",
		"owner-name",
		"owner-id",
		"owner-me",
		"group-name",
		"group-id",
		"group-me",
		"readable",
		"writable",
		"executable",
	} {
		suite.Run(name+" wrong argument type", func() {
			// Arrange
			var typeError *utils.ArgumentTypeError

			// Act
			value, err := values.NewFilepath(nil, map[string][]any{
				name: {1.5},
			})

			// Assert
			suite.Nil(value)
			suite.Require().ErrorAs(err, &typeError)
		})
	}

	// Act
	value, err := values.New("filepath", map[string]any{
		"unknown": true,
	}, nil)
	unknownType, typeError := values.NewFilepath(nil, map[string][]any{
		"type": {"unknown"},
	})

	// Assert
	suite.Nil(value)
	suite.Require().ErrorIs(err, values.ErrNoProperties)
	suite.Nil(unknownType)
	suite.Require().ErrorIs(typeError, values.ErrFilepathUnknownType)
}

func (suite *FilepathTestSuite) TestOwnershipAndAccess() {
	// Arrange
	file := filepath.Join(suite.T().TempDir(), "привет")
	suite.Require().NoError(os.WriteFile(file, nil, 0o700))
	suite.Require().NoError(os.Chmod(file, 0o700))

	usr, err := user.Current()
	suite.Require().NoError(err)
	uid, err := strconv.ParseInt(usr.Uid, 10, 64)
	suite.Require().NoError(err)
	info, err := os.Stat(file)
	suite.Require().NoError(err)

	stat, ok := info.Sys().(*syscall.Stat_t)
	suite.Require().True(ok)

	grp, err := user.LookupGroupId(strconv.FormatUint(uint64(stat.Gid), 10))
	suite.Require().NoError(err)

	for name, argument := range map[string]any{
		"owner-name": usr.Username,
		"owner-id":   uid,
		"owner-me":   true,
		"group-name": grp.Name,
		"group-id":   int64(stat.Gid),
		"group-me":   true,
		"readable":   true,
		"writable":   true,
		"executable": true,
	} {
		suite.Run(name, func() {
			// Arrange
			value, err := values.NewFilepath(nil, map[string][]any{
				name: {argument},
			})
			suite.Require().NoError(err)

			// Act
			err = value.Validate(file)
			missingError := value.Validate(file + "-missing")

			// Assert
			suite.Require().NoError(err)
			suite.Require().ErrorIs(missingError, fs.ErrNotExist)
		})
	}

	for name, want := range map[string]error{
		"readable":   values.ErrFilepathIsReadable,
		"writable":   values.ErrFilepathIsWritable,
		"executable": values.ErrFilepathIsExecutable,
	} {
		suite.Run(name+" false", func() {
			// Arrange
			value, err := values.NewFilepath(nil, map[string][]any{
				name: {false},
			})
			suite.Require().NoError(err)

			// Act
			err = value.Validate(file)

			// Assert
			suite.Require().ErrorIs(err, want)
		})
	}

	for _, name := range []string{
		"owner-me",
		"group-me",
	} {
		// Arrange
		value, err := values.NewFilepath(nil, map[string][]any{
			name: {false},
		})
		suite.Require().NoError(err)

		// Act
		err = value.Validate(file)

		// Assert
		suite.Require().Error(err)
	}
}

func (suite *FilepathTestSuite) TestCompletionErrorsAndFiltering() {
	// Arrange
	root := suite.T().TempDir()
	for _, name := range []string{
		"a.txt",
		"ab.txt",
		"ba.txt",
		"a.json",
	} {
		suite.Require().NoError(os.WriteFile(filepath.Join(root, name), nil, 0o600))
	}

	value, err := values.New("filepath", nil, nil)
	suite.Require().NoError(err)

	// Act
	paths, directive := value.Complete(filepath.Join(root, "a"))
	missing, missingDirective := value.Complete(filepath.Join(root, "missing", "a"))

	// Assert
	suite.Equal([]string{
		filepath.Join(root, "a.json"),
		filepath.Join(root, "a.txt"),
		filepath.Join(root, "ab.txt"),
	}, paths)
	suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
	suite.Nil(missing)
	suite.Equal(cobra.ShellCompDirectiveError, missingDirective)
}

func (suite *FilepathTestSuite) TestConfigurationBoundaries() {
	for _, test := range []struct {
		name    string
		minimum int64
		maximum int64
	}{
		{
			name:    "owner-id",
			maximum: math.MaxUint32,
		},
		{
			name:    "group-id",
			maximum: math.MaxUint32,
		},
		{
			name:    "permissions",
			maximum: 0o777,
		},
	} {
		suite.Run(test.name, func() {
			for _, bound := range []int64{test.minimum, test.maximum} {
				suite.Run(strconv.FormatInt(bound, 10), func() {
					// Arrange
					properties := map[string][]any{
						test.name: {bound},
					}

					// Act
					value, err := values.NewFilepath(nil, properties)

					// Assert
					suite.Require().NoError(err)
					suite.NotNil(value)
				})
			}
		})
	}
}

func (suite *FilepathTestSuite) TestExecutePermissionDenied() {
	// Arrange
	file := filepath.Join(suite.T().TempDir(), "привет")
	suite.Require().NoError(os.WriteFile(file, nil, 0o600))
	suite.Require().NoError(os.Chmod(file, 0o600))

	for _, expected := range []bool{true, false} {
		suite.Run(strconv.FormatBool(expected), func() {
			// Arrange
			value, err := values.NewFilepath(nil, map[string][]any{
				"executable": {expected},
			})
			suite.Require().NoError(err)

			// Act
			err = value.Validate(file)

			// Assert
			if expected {
				suite.Require().ErrorIs(err, values.ErrFilepathIsNotExecutable)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func TestFilepath(t *testing.T) {
	t.Parallel()

	suite.Run(t, &FilepathTestSuite{})
}
