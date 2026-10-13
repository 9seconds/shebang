package utils

import (
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type WorldTestSuite struct {
	suite.Suite
}

func (suite *WorldTestSuite) TestStatAndLstat() {
	// Arrange
	root := suite.T().TempDir()
	file := filepath.Join(root, "привет")
	link := filepath.Join(root, "link")
	missing := filepath.Join(root, "missing")
	dangling := filepath.Join(root, "dangling")

	suite.Require().NoError(os.WriteFile(file, []byte("привет"), 0o600))
	suite.Require().NoError(os.Symlink(file, link))
	suite.Require().NoError(os.Symlink(missing, dangling))

	for _, test := range []struct {
		name string
		path string
		mode fs.FileMode
	}{
		{
			name: "regular file",
			path: file,
		},
		{
			name: "directory",
			path: root,
			mode: fs.ModeDir,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			var world worldView

			for _, stat := range []func(string) (fs.FileInfo, error){world.Stat, world.Lstat} {
				// Act
				info, err := stat(test.path)
				missingInfo, missingError := stat(missing)

				// Assert
				suite.Require().NoError(err)
				suite.Equal(test.mode, info.Mode().Type())
				suite.Require().ErrorIs(missingError, fs.ErrNotExist)
				suite.Nil(missingInfo)
			}
		})
	}

	for _, test := range []struct {
		name  string
		path  string
		lstat bool
		mode  fs.FileMode
		want  error
	}{
		{
			name: "follow link",
			path: link,
		},
		{
			name:  "inspect link",
			path:  link,
			lstat: true,
			mode:  fs.ModeSymlink,
		},
		{
			name: "follow dangling link",
			path: dangling,
			want: fs.ErrNotExist,
		},
		{
			name:  "inspect dangling link",
			path:  dangling,
			lstat: true,
			mode:  fs.ModeSymlink,
		},
	} {
		suite.Run(test.name, func() {
			// Arrange
			var world worldView

			stat := world.Stat
			if test.lstat {
				stat = world.Lstat
			}

			// Act
			info, err := stat(test.path)

			// Assert
			suite.Require().ErrorIs(err, test.want)

			if test.want != nil {
				suite.Nil(info)
			} else {
				suite.Equal(test.mode, info.Mode().Type())
			}
		})
	}
}

func (suite *WorldTestSuite) TestStatCachesResults() {
	for _, name := range []string{
		"stat",
		"lstat",
	} {
		suite.Run(name, func() {
			// Arrange
			var world worldView

			stat := world.Stat
			if name == "lstat" {
				stat = world.Lstat
			}

			file := filepath.Join(suite.T().TempDir(), "привет")
			suite.Require().NoError(os.WriteFile(file, []byte("old"), 0o600))
			before, err := stat(file)
			suite.Require().NoError(err)
			suite.Require().NoError(os.Remove(file))

			// Act
			after, err := stat(file)

			// Assert
			suite.Require().NoError(err)
			suite.Same(before, after)

			// Arrange
			missing := file + "-missing"
			_, original := stat(missing)
			suite.Require().ErrorIs(original, fs.ErrNotExist)
			suite.Require().NoError(os.WriteFile(missing, nil, 0o600))

			// Act
			info, err := stat(missing)

			// Assert
			suite.Nil(info)
			suite.Same(original, err)
		})
	}
}

func (suite *WorldTestSuite) TestIdentities() {
	// Arrange
	var world worldView

	usr, err := user.Current()
	suite.Require().NoError(err)
	uid, err := strconv.ParseUint(usr.Uid, 10, 32)
	suite.Require().NoError(err)

	// Act
	current, err := world.CurrentUser()

	// Assert
	suite.Require().NoError(err)
	suite.Equal(uint32(uid), current)

	// Act
	named, err := world.LookupUser(usr.Username)

	// Assert
	suite.Require().NoError(err)
	suite.Equal(current, named)

	// Act
	named, err = world.LookupUser(usr.Username)

	// Assert
	suite.Require().NoError(err)
	suite.Equal(current, named)

	// Arrange
	grp, err := user.LookupGroupId(usr.Gid)
	suite.Require().NoError(err)
	gid, err := strconv.ParseUint(grp.Gid, 10, 32)
	suite.Require().NoError(err)

	// Act
	group, err := world.LookupGroup(grp.Name)

	// Assert
	suite.Require().NoError(err)
	suite.Equal(uint32(gid), group)

	// Act
	group, err = world.LookupGroup(grp.Name)

	// Assert
	suite.Require().NoError(err)
	suite.Equal(uint32(gid), group)

	// Arrange
	groupIDs, err := usr.GroupIds()
	suite.Require().NoError(err)

	expected := make([]uint32, 0, len(groupIDs))

	for _, id := range groupIDs {
		parsed, err := strconv.ParseUint(id, 10, 32)
		suite.Require().NoError(err)

		expected = append(expected, uint32(parsed))
	}

	// Act
	groups, err := world.CurrentGroups()

	// Assert
	suite.Require().NoError(err)
	suite.ElementsMatch(expected, groups)

	// Act
	groups, err = world.CurrentGroups()

	// Assert
	suite.Require().NoError(err)
	suite.ElementsMatch(expected, groups)
}

func (suite *WorldTestSuite) TestIdentityErrorsAreCached() {
	// Arrange
	var world worldView

	for _, lookup := range []func(string) (uint32, error){world.LookupUser, world.LookupGroup} {
		// Act
		id, original := lookup("shebang-test-nonexistent-identity-7f89c2")
		cachedID, err := lookup("shebang-test-nonexistent-identity-7f89c2")

		// Assert
		suite.Require().Error(original)
		suite.Zero(id)
		suite.Zero(cachedID)
		suite.Equal(original, err)
	}
}

func (suite *WorldTestSuite) TestAccessAndCaching() {
	// Arrange
	var world worldView

	file := filepath.Join(suite.T().TempDir(), "привет")
	suite.Require().NoError(os.WriteFile(file, nil, 0o700))
	suite.Require().NoError(os.Chmod(file, 0o700))

	accesses := []func(string) error{world.IsReadable, world.IsWritable, world.IsExecutable}

	for _, access := range accesses {
		// Act
		err := access(file)
		missingError := access(file + "-missing")

		// Assert
		suite.Require().NoError(err)
		suite.Require().ErrorIs(missingError, fs.ErrNotExist)
	}

	suite.Require().NoError(os.Remove(file))

	for _, access := range accesses {
		// Act
		err := access(file)

		// Assert
		suite.Require().NoError(err)
	}

	// Arrange
	denied := filepath.Join(suite.T().TempDir(), "not-executable")
	suite.Require().NoError(os.WriteFile(denied, nil, 0o600))
	suite.Require().NoError(os.Chmod(denied, 0o600))
	suite.Require().ErrorIs(world.IsExecutable(denied), fs.ErrPermission)
	suite.Require().NoError(world.IsReadable(denied))
	suite.Require().NoError(os.Chmod(denied, 0o700))

	// Act
	err := world.IsExecutable(denied)

	// Assert
	suite.Require().ErrorIs(err, fs.ErrPermission)
}

func (suite *WorldTestSuite) TestConcurrentReads() {
	// Arrange
	var world worldView

	file := filepath.Join(suite.T().TempDir(), "привет")
	suite.Require().NoError(os.WriteFile(file, nil, 0o700))
	suite.Require().NoError(os.Chmod(file, 0o700))

	errors := make(chan error, 32*5)

	var workers sync.WaitGroup

	// Act
	for range 32 {
		workers.Go(func() {
			_, err := world.Stat(file)
			errors <- err

			_, err = world.Lstat(file)
			errors <- err

			errors <- world.IsReadable(file)

			_, err = world.CurrentUser()
			errors <- err

			_, err = world.CurrentGroups()
			errors <- err
		})
	}

	workers.Wait()
	close(errors)

	// Assert
	for err := range errors {
		suite.Require().NoError(err)
	}
}

func TestWorld(t *testing.T) {
	t.Parallel()

	suite.Run(t, &WorldTestSuite{})
}
