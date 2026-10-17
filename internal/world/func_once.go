package world

import "sync"

type funcOnce[R any] struct {
	sync.Once

	result R
	err    error
}

//nolint:ireturn // Preserve the caller's generic result type.
func (f *funcOnce[R]) Do(cb func() R) R {
	f.Once.Do(func() {
		f.result = cb()
	})

	return f.result
}

//nolint:ireturn // Preserve the caller's generic result type.
func (f *funcOnce[R]) DoE(cb func() (R, error)) (R, error) {
	f.Once.Do(func() {
		f.result, f.err = cb()
	})

	return f.result, f.err
}
