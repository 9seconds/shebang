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
			var world worldView

			for _, stat := range []func(string) (fs.FileInfo, error){world.Stat, world.Lstat} {
				info, err := stat(test.path)
				suite.Require().NoError(err)
				suite.Equal(test.mode, info.Mode().Type())
				info, err = stat(missing)
				suite.Require().ErrorIs(err, fs.ErrNotExist)
				suite.Nil(info)
			}
		})
	}

	var world worldView

	info, err := world.Stat(link)
	suite.Require().NoError(err)
	suite.True(info.Mode().IsRegular())
	info, err = world.Lstat(link)
	suite.Require().NoError(err)
	suite.Equal(fs.ModeSymlink, info.Mode().Type())
	info, err = world.Stat(dangling)
	suite.Require().ErrorIs(err, fs.ErrNotExist)
	suite.Nil(info)
	info, err = world.Lstat(dangling)
	suite.Require().NoError(err)
	suite.Equal(fs.ModeSymlink, info.Mode().Type())
}

func (suite *WorldTestSuite) TestStatCachesResults() {
	for _, name := range []string{"stat", "lstat"} {
		suite.Run(name, func() {
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
			after, err := stat(file)
			suite.Require().NoError(err)
			suite.Same(before, after)

			missing := file + "-missing"
			_, original := stat(missing)
			suite.Require().ErrorIs(original, fs.ErrNotExist)
			suite.Require().NoError(os.WriteFile(missing, nil, 0o600))
			info, err := stat(missing)
			suite.Nil(info)
			suite.Same(original, err)
		})
	}
}

func (suite *WorldTestSuite) TestIdentities() {
	var world worldView

	usr, err := user.Current()
	suite.Require().NoError(err)
	uid, err := strconv.ParseUint(usr.Uid, 10, 32)
	suite.Require().NoError(err)
	current, err := world.CurrentUser()
	suite.Require().NoError(err)
	suite.Equal(uint32(uid), current)

	named, err := world.LookupUser(usr.Username)
	suite.Require().NoError(err)
	suite.Equal(current, named)
	named, err = world.LookupUser(usr.Username)
	suite.Require().NoError(err)
	suite.Equal(current, named)

	grp, err := user.LookupGroupId(usr.Gid)
	suite.Require().NoError(err)
	gid, err := strconv.ParseUint(grp.Gid, 10, 32)
	suite.Require().NoError(err)
	group, err := world.LookupGroup(grp.Name)
	suite.Require().NoError(err)
	suite.Equal(uint32(gid), group)
	group, err = world.LookupGroup(grp.Name)
	suite.Require().NoError(err)
	suite.Equal(uint32(gid), group)

	groupIDs, err := usr.GroupIds()
	suite.Require().NoError(err)

	expected := make([]uint32, 0, len(groupIDs))

	for _, id := range groupIDs {
		parsed, err := strconv.ParseUint(id, 10, 32)
		suite.Require().NoError(err)

		expected = append(expected, uint32(parsed))
	}

	groups, err := world.CurrentGroups()
	suite.Require().NoError(err)
	suite.ElementsMatch(expected, groups)
	groups, err = world.CurrentGroups()
	suite.Require().NoError(err)
	suite.ElementsMatch(expected, groups)
}

func (suite *WorldTestSuite) TestIdentityErrorsAreCached() {
	var world worldView

	for _, lookup := range []func(string) (uint32, error){world.LookupUser, world.LookupGroup} {
		id, original := lookup("shebang-test-nonexistent-identity-7f89c2")
		suite.Require().Error(original)
		suite.Zero(id)
		id, err := lookup("shebang-test-nonexistent-identity-7f89c2")
		suite.Zero(id)
		suite.Equal(original, err)
	}
}

func (suite *WorldTestSuite) TestAccessAndCaching() {
	var world worldView

	file := filepath.Join(suite.T().TempDir(), "привет")
	suite.Require().NoError(os.WriteFile(file, nil, 0o700))
	suite.Require().NoError(os.Chmod(file, 0o700))

	accesses := []func(string) error{world.IsReadable, world.IsWritable, world.IsExecutable}

	for _, access := range accesses {
		suite.Require().NoError(access(file))
		suite.Require().ErrorIs(access(file+"-missing"), fs.ErrNotExist)
	}

	suite.Require().NoError(os.Remove(file))

	for _, access := range accesses {
		suite.Require().NoError(access(file))
	}

	denied := filepath.Join(suite.T().TempDir(), "not-executable")
	suite.Require().NoError(os.WriteFile(denied, nil, 0o600))
	suite.Require().NoError(os.Chmod(denied, 0o600))
	suite.Require().ErrorIs(world.IsExecutable(denied), fs.ErrPermission)
	suite.Require().NoError(world.IsReadable(denied))
	suite.Require().NoError(os.Chmod(denied, 0o700))
	suite.Require().ErrorIs(world.IsExecutable(denied), fs.ErrPermission)
}

func (suite *WorldTestSuite) TestConcurrentReads() {
	var world worldView

	file := filepath.Join(suite.T().TempDir(), "привет")
	suite.Require().NoError(os.WriteFile(file, nil, 0o700))
	suite.Require().NoError(os.Chmod(file, 0o700))

	errors := make(chan error, 32*5)

	var workers sync.WaitGroup

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

	for err := range errors {
		suite.Require().NoError(err)
	}
}

func TestWorld(t *testing.T) {
	t.Parallel()

	suite.Run(t, &WorldTestSuite{})
}
