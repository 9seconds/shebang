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
	root := suite.T().TempDir()
	directory := filepath.Join(root, "привет")
	suite.Require().NoError(os.Mkdir(directory, 0o755))
	suite.Require().NoError(os.WriteFile(filepath.Join(directory, "report.txt"), nil, 0o600))
	suite.Require().NoError(os.WriteFile(filepath.Join(directory, "report.json"), nil, 0o600))

	value, err := values.NewFilepath(nil, map[string][]any{
		"type":            {"regular"},
		"file-extensions": {"txt"},
	})
	suite.Require().NoError(err)
	suite.Require().Error(value.Validate(directory))

	for _, test := range []struct {
		name      string
		input     string
		want      []string
		directive cobra.ShellCompDirective
	}{
		{
			name:      "directory despite file constraints",
			input:     root + string(os.PathSeparator),
			want:      []string{directory + string(os.PathSeparator)},
			directive: cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace,
		},
		{
			name:      "unicode directory prefix",
			input:     filepath.Join(root, "при"),
			want:      []string{directory + string(os.PathSeparator)},
			directive: cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace,
		},
		{
			name:      "files inside directory remain filtered",
			input:     filepath.Join(directory, "rep"),
			want:      []string{filepath.Join(directory, "report.txt")},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
	} {
		suite.Run(test.name, func() {
			paths, directive := value.Complete(test.input)
			suite.Equal(test.want, paths)
			suite.Equal(test.directive, directive)
		})
	}
}

func (suite *FilepathTestSuite) TestTypes() {
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
			value, err := values.New("filepath", nil, map[string][]any{"type": {test.name}})
			suite.Require().NoError(err)
			suite.Require().NoError(value.Validate(test.accepted))
			suite.Require().ErrorIs(value.Validate(test.rejected), values.ErrFilepathNoSuitableType)
		})
	}

	value, err := values.NewFilepath(nil, map[string][]any{
		"type": {"regular", "directory", "regular"},
	})
	suite.Require().NoError(err)
	suite.Equal("filepath(type:directory,regular)", value.String())
	suite.Require().NoError(value.Validate(file))
	suite.Require().NoError(value.Validate(root))
}

func (suite *FilepathTestSuite) TestPathsAndConstraints() {
	root := suite.T().TempDir()
	file := filepath.Join(root, " привет.txt ")
	suite.Require().NoError(os.WriteFile(file, nil, 0o600))
	suite.Require().NoError(os.Chmod(file, 0o600))

	value, err := values.NewFilepath(nil, nil)
	suite.Require().NoError(err)
	suite.Require().ErrorIs(value.Validate(""), values.ErrFilepathNotSet)
	suite.Require().NoError(value.Validate("привет"))
	suite.Require().NoError(value.Validate(" "))

	value, err = values.NewFilepath(nil, map[string][]any{"type": {"exists"}})
	suite.Require().NoError(err)
	suite.Require().NoError(value.Validate(file))
	suite.Require().Error(value.Validate(filepath.Join(root, "привет.txt")))

	value, err = values.NewFilepath(nil, map[string][]any{"root": {root}})
	suite.Require().NoError(err)
	suite.Require().NoError(value.Validate(root))
	suite.Require().NoError(value.Validate(filepath.Join(root, "new.txt")))
	suite.Require().Error(value.Validate(root + "-other/new.txt"))
	suite.Require().Error(value.Validate(filepath.Join(root, "..", "outside")))

	value, err = values.NewFilepath(nil, map[string][]any{"file-extensions": {"txt", ".json", "txt"}})
	suite.Require().NoError(err)
	suite.Equal("filepath(file-extensions:.json,.txt)", value.String())
	suite.Require().NoError(value.Validate("привет.txt"))
	suite.Require().NoError(value.Validate("report.json"))
	suite.Require().Error(value.Validate("report.TXT"))
	suite.Require().Error(value.Validate("report.txt.gz"))

	value, err = values.NewFilepath(nil, map[string][]any{"permissions": {int64(0o600)}})
	suite.Require().NoError(err)
	suite.Require().NoError(value.Validate(file))
	value, err = values.NewFilepath(nil, map[string][]any{"permissions": {int64(0o644)}})
	suite.Require().NoError(err)
	suite.Require().ErrorContains(value.Validate(file), "incorrect permissions")
	suite.Require().ErrorIs(value.Validate(filepath.Join(root, "missing")), fs.ErrNotExist)
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
			name: "root",
			want: utils.ErrEmpty,
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
			name:      "group-id",
			arguments: []any{int64(math.MaxUint32) + 1},
			want:      values.ErrFilepathIncorrectGID,
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewFilepath(nil, map[string][]any{test.name: test.arguments})
			suite.Nil(value)
			suite.Require().ErrorIs(err, test.want)
		})
	}

	for _, name := range []string{
		"root", "type", "file-extensions", "permissions",
		"owner-name", "owner-id", "owner-me", "group-name", "group-id", "group-me",
		"readable", "writable", "executable",
	} {
		suite.Run(name+" wrong argument type", func() {
			value, err := values.NewFilepath(nil, map[string][]any{name: {1.5}})
			suite.Nil(value)

			var typeError *utils.ArgumentTypeError

			suite.Require().ErrorAs(err, &typeError)
		})
	}

	value, err := values.New("filepath", map[string]any{"unknown": true}, nil)
	suite.Nil(value)
	suite.Require().ErrorIs(err, values.ErrNoProperties)
	value, err = values.NewFilepath(nil, map[string][]any{"type": {"unknown"}})
	suite.Nil(value)
	suite.Require().ErrorContains(err, "unknown type unknown")
}

func (suite *FilepathTestSuite) TestOwnershipAndAccess() {
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
			value, err := values.NewFilepath(nil, map[string][]any{name: {argument}})
			suite.Require().NoError(err)
			suite.Require().NoError(value.Validate(file))
			suite.Require().ErrorIs(value.Validate(file+"-missing"), fs.ErrNotExist)
		})
	}

	for name, want := range map[string]error{
		"readable":   values.ErrFilepathIsReadable,
		"writable":   values.ErrFilepathIsWritable,
		"executable": values.ErrFilepathIsExecutable,
	} {
		suite.Run(name+" false", func() {
			value, err := values.NewFilepath(nil, map[string][]any{name: {false}})
			suite.Require().NoError(err)
			suite.Require().ErrorIs(value.Validate(file), want)
		})
	}

	for _, name := range []string{"owner-me", "group-me"} {
		value, err := values.NewFilepath(nil, map[string][]any{name: {false}})
		suite.Require().NoError(err)
		suite.Require().Error(value.Validate(file))
	}
}

func (suite *FilepathTestSuite) TestCompletionErrorsAndFiltering() {
	root := suite.T().TempDir()
	for _, name := range []string{"a.txt", "ab.txt", "ba.txt", "a.json"} {
		suite.Require().NoError(os.WriteFile(filepath.Join(root, name), nil, 0o600))
	}

	value, err := values.New("filepath", nil, map[string][]any{"file-extensions": {"txt"}})
	suite.Require().NoError(err)

	paths, directive := value.Complete(filepath.Join(root, "a"))
	suite.Equal([]string{filepath.Join(root, "a.txt"), filepath.Join(root, "ab.txt")}, paths)
	suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
	paths, directive = value.Complete(filepath.Join(root, "missing", "a"))
	suite.Nil(paths)
	suite.Equal(cobra.ShellCompDirectiveError, directive)
}

func TestFilepath(t *testing.T) {
	t.Parallel()

	suite.Run(t, &FilepathTestSuite{})
}
