package world

import (
	"io/fs"
	"os"
	"syscall"

	"github.com/9seconds/shebang/internal/log"
	"golang.org/x/sys/unix"
)

// Files caches filesystem metadata and access checks, including lookup errors.
type Files struct {
	stats      memoized[string, fs.FileInfo]
	lstats     memoized[string, fs.FileInfo]
	readable   memoized[string, error]
	writable   memoized[string, error]
	executable memoized[string, error]
}

// Stat returns cached metadata, following symbolic links.
func (f *Files) Stat(path string) (fs.FileInfo, error) {
	return f.stats.DoE(path, func() (fs.FileInfo, error) {
		info, err := os.Stat(path)
		log.Printf("Files: stat %q (following symlinks): %v", path, err)

		return info, err
	})
}

// Lstat returns cached metadata for the path itself, without following links.
func (f *Files) Lstat(path string) (fs.FileInfo, error) {
	return f.lstats.DoE(path, func() (fs.FileInfo, error) {
		info, err := os.Lstat(path)
		log.Printf("Files: lstat %q (inspecting link): %v", path, err)

		return info, err
	})
}

// Readable checks read access using OS access rules and caches the result.
func (f *Files) Readable(path string) error {
	return f.readable.Do(path, func() error {
		err := syscall.Access(path, unix.R_OK)
		log.Printf("Files: read access to %q allowed=%t (error: %v)", path, err == nil, err)

		return err
	})
}

// Writable checks write access using OS access rules and caches the result.
func (f *Files) Writable(path string) error {
	return f.writable.Do(path, func() error {
		err := syscall.Access(path, unix.W_OK)
		log.Printf("Files: write access to %q allowed=%t (error: %v)", path, err == nil, err)

		return err
	})
}

// Executable checks execute/search access using OS rules and caches the result.
func (f *Files) Executable(path string) error {
	return f.executable.Do(path, func() error {
		err := syscall.Access(path, unix.X_OK)
		log.Printf("Files: execute/search access to %q allowed=%t (error: %v)", path, err == nil, err)

		return err
	})
}
