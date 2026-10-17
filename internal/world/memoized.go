package world

import "sync"

type memoized[A comparable, R any] struct {
	results sync.Map
}

//nolint:ireturn // Preserve the caller's generic result type.
func (m *memoized[A, R]) Do(parameter A, cb func() R) R {
	val, _ := m.results.LoadOrStore(parameter, &funcOnce[R]{})
	value, _ := val.(*funcOnce[R])

	return value.Do(cb)
}

//nolint:ireturn // Preserve the caller's generic result type.
func (m *memoized[A, R]) DoE(parameter A, cb func() (R, error)) (R, error) {
	val, _ := m.results.LoadOrStore(parameter, &funcOnce[R]{})
	value, _ := val.(*funcOnce[R])

	return value.DoE(cb)
}
