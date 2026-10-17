package world

import (
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sys/unix"
)

type FilesTestSuite struct {
	suite.Suite
}

func (s *FilesTestSuite) TestStat() {
	t := s.T()

	// Arrange
	root := t.TempDir()
	file := filepath.Join(root, "привет")
	link := filepath.Join(root, "link")
	dangling := filepath.Join(root, "dangling")

	require.NoError(t, os.WriteFile(file, []byte("hello"), 0o600))
	require.NoError(t, os.Symlink(file, link))
	require.NoError(t, os.Symlink(filepath.Join(root, "absent"), dangling))

	for _, test := range []struct {
		name string
		path string
		mode fs.FileMode
	}{
		{
			name: "regular",
			path: file,
		},
		{
			name: "directory",
			path: root,
			mode: fs.ModeDir,
		},
		{
			name: "symlink",
			path: link,
			mode: fs.ModeSymlink,
		},
	} {
		s.Run(test.name, func() {
			t := s.T()

			// Arrange
			var files Files

			// Act
			info, err := files.Lstat(test.path)
			followed, statError := files.Stat(test.path)

			// Assert
			require.NoError(t, err)
			require.Equal(t, test.mode, info.Mode().Type())

			require.NoError(t, statError)

			if test.mode == fs.ModeSymlink {
				require.True(t, followed.Mode().IsRegular())
			}
		})
	}

	// Act
	var files Files

	info, err := files.Lstat(dangling)
	missing, statError := files.Stat(dangling)

	// Assert
	require.NoError(t, err)
	require.Equal(t, fs.ModeSymlink, info.Mode().Type())

	require.Nil(t, missing)
	require.ErrorIs(t, statError, fs.ErrNotExist)
}

func (s *FilesTestSuite) TestCaches() {
	t := s.T()

	for _, lstat := range []bool{false, true} {
		// Arrange
		var files Files

		path := filepath.Join(t.TempDir(), "cached")

		stat := files.Stat
		if lstat {
			stat = files.Lstat
		}

		// Act
		_, firstError := stat(path)

		require.NoError(t, os.WriteFile(path, []byte("data"), 0o600))

		_, cachedError := stat(path)

		var fresh Files

		info, freshError := fresh.Stat(path)

		require.NoError(t, os.Remove(path))

		cachedInfo, successError := fresh.Stat(path)

		// Assert
		require.ErrorIs(t, firstError, fs.ErrNotExist)
		require.Same(t, firstError, cachedError)

		require.NoError(t, freshError)

		require.NoError(t, successError)
		require.Same(t, info, cachedInfo)
	}
}

func (s *FilesTestSuite) TestSpecialTypes() {
	t := s.T()

	// Arrange
	var files Files

	root := t.TempDir()
	pipe := filepath.Join(root, "pipe")
	socket := filepath.Join(root, "socket")

	require.NoError(t, syscall.Mkfifo(pipe, 0o600))

	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	// Act
	pipeInfo, pipeError := files.Stat(pipe)
	socketInfo, socketError := files.Stat(socket)
	deviceInfo, deviceError := files.Stat("/dev/null")

	// Assert
	require.NoError(t, pipeError)
	require.Equal(t, fs.ModeNamedPipe, pipeInfo.Mode().Type())

	require.NoError(t, socketError)
	require.Equal(t, fs.ModeSocket, socketInfo.Mode().Type())

	require.NoError(t, deviceError)
	require.Equal(t, fs.ModeDevice|fs.ModeCharDevice, deviceInfo.Mode().Type())
}

func (s *FilesTestSuite) TestAccessMissingCached() {
	for _, name := range []string{
		"read",
		"write",
		"execute",
	} {
		s.Run(name, func() {
			t := s.T()

			// Arrange
			var files Files

			path := filepath.Join(t.TempDir(), "missing")
			checks := map[string]func(string) error{
				"read":    files.Readable,
				"write":   files.Writable,
				"execute": files.Executable,
			}
			check := checks[name]

			// Act
			first := check(path)

			require.NoError(t, os.WriteFile(path, nil, 0o700))

			cached := check(path)

			// Assert
			require.ErrorIs(t, first, fs.ErrNotExist)
			require.Equal(t, first, cached)
		})
	}
}

func (s *FilesTestSuite) TestAccess() {
	t := s.T()

	for _, mode := range []fs.FileMode{0o000, 0o400, 0o200, 0o100, 0o700} {
		// Arrange
		var files Files

		path := filepath.Join(t.TempDir(), "access")
		require.NoError(t, os.WriteFile(path, nil, mode))

		for _, test := range []struct {
			check func(string) error
			mask  uint32
		}{
			{
				check: files.Readable,
				mask:  unix.R_OK,
			},
			{
				check: files.Writable,
				mask:  unix.W_OK,
			},
			{
				check: files.Executable,
				mask:  unix.X_OK,
			},
		} {
			// Act
			want := syscall.Access(path, test.mask)
			got := test.check(path)

			require.NoError(t, os.Remove(path))

			cached := test.check(path)

			require.NoError(t, os.WriteFile(path, nil, mode))

			// Assert
			require.Equal(t, want, got)
			require.Equal(t, got, cached)
		}
	}
}

func (s *FilesTestSuite) TestConcurrent() {
	t := s.T()

	// Arrange
	var (
		files   Files
		workers sync.WaitGroup
	)

	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o700))

	errs := make([]error, 32)

	// Act
	for idx := range errs {
		workers.Go(func() {
			_, errs[idx] = files.Stat(path)
			_ = files.Readable(path)
			_ = files.Writable(path)
			_ = files.Executable(path)
		})
	}

	workers.Wait()

	// Assert
	for _, err := range errs {
		require.NoError(t, err)
	}
}

func TestFiles(t *testing.T) {
	t.Parallel()

	suite.Run(t, &FilesTestSuite{})
}
