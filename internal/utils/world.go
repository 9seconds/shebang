package utils

import (
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type worldViewResult[T any] struct {
	sync.Once

	result T
	err    error
}

type worldView struct {
	fileInfos  sync.Map
	lFileInfos sync.Map
	users      sync.Map
	groups     sync.Map
	accesses   sync.Map

	currentGroups worldViewResult[[]uint32]
}

func (w *worldView) Stat(path string) (fs.FileInfo, error) {
	val, _ := w.fileInfos.LoadOrStore(path, &worldViewResult[fs.FileInfo]{})
	value, _ := val.(*worldViewResult[fs.FileInfo])

	value.Do(func() {
		value.result, value.err = os.Stat(path)
	})

	return value.result, value.err
}

func (w *worldView) Lstat(path string) (fs.FileInfo, error) {
	val, _ := w.lFileInfos.LoadOrStore(path, &worldViewResult[fs.FileInfo]{})
	value, _ := val.(*worldViewResult[fs.FileInfo])

	value.Do(func() {
		value.result, value.err = os.Lstat(path)
	})

	return value.result, value.err
}

func (w *worldView) CurrentUser() (uint32, error) {
	return w.LookupUser("")
}

func (w *worldView) CurrentGroups() ([]uint32, error) {
	w.currentGroups.Do(func() {
		usr, err := user.Current()
		if err != nil {
			w.currentGroups.err = err

			return
		}

		groups, err := usr.GroupIds()
		if err != nil {
			w.currentGroups.err = err

			return
		}

		w.currentGroups.result = make([]uint32, len(groups))

		for idx, strGid := range groups {
			value, err := strconv.ParseUint(strGid, 10, 32)
			if err != nil {
				panic(err)
			}

			w.currentGroups.result[idx] = uint32(value)
		}
	})

	return w.currentGroups.result, w.currentGroups.err
}

func (w *worldView) LookupUser(name string) (uint32, error) {
	val, _ := w.users.LoadOrStore(name, &worldViewResult[uint32]{})
	value, _ := val.(*worldViewResult[uint32])

	value.Do(func() {
		var (
			usr *user.User
			err error
		)

		if name == "" {
			usr, err = user.Current()
		} else {
			usr, err = user.Lookup(name)
		}

		if err != nil {
			value.err = err

			return
		}

		uid, err := strconv.ParseUint(usr.Uid, 10, 32)
		if err != nil {
			panic(err)
		}

		value.result = uint32(uid)
	})

	return value.result, value.err
}

func (w *worldView) LookupGroup(name string) (uint32, error) {
	val, _ := w.groups.LoadOrStore(name, &worldViewResult[uint32]{})
	value, _ := val.(*worldViewResult[uint32])

	value.Do(func() {
		grp, err := user.LookupGroup(name)
		if err != nil {
			value.err = err

			return
		}

		rv, err := strconv.ParseUint(grp.Gid, 10, 32)
		value.result, value.err = uint32(rv), err
	})

	return value.result, value.err
}

func (w *worldView) IsReadable(path string) error {
	return w.isAccess(path, unix.R_OK)
}

func (w *worldView) IsWritable(path string) error {
	return w.isAccess(path, unix.W_OK)
}

func (w *worldView) IsExecutable(path string) error {
	return w.isAccess(path, unix.X_OK)
}

func (w *worldView) isAccess(path string, mask uint32) error {
	key := path + strconv.FormatUint(uint64(mask), 10)
	val, _ := w.accesses.LoadOrStore(key, &worldViewResult[error]{})
	value, _ := val.(*worldViewResult[error])

	value.Do(func() {
		value.err = syscall.Access(path, mask)
	})

	return value.err
}

// World caches filesystem and identity queries for the current process.
var World = worldView{}
