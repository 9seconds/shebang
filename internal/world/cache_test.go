package world

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CacheTestSuite struct {
	suite.Suite
}

func (s *CacheTestSuite) TestFuncOnce() {
	for _, fail := range []bool{false, true} {
		names := map[bool]string{
			false: "success",
			true:  "error",
		}
		s.Run(names[fail], func() {
			t := s.T()

			// Arrange
			var (
				cached  funcOnce[int]
				calls   atomic.Int32
				workers sync.WaitGroup
			)

			wantError := errors.New("cached failure")
			results := make([]int, 32)
			errs := make([]error, len(results))

			// Act
			for idx := range results {
				workers.Go(func() {
					results[idx], errs[idx] = cached.DoE(func() (int, error) {
						calls.Add(1)

						if fail {
							return 42, wantError
						}

						return 42, nil
					})
				})
			}

			workers.Wait()

			// Assert
			require.EqualValues(t, 1, calls.Load())

			for idx, result := range results {
				require.Equal(t, 42, result)

				if fail {
					require.ErrorIs(t, errs[idx], wantError)
				} else {
					require.NoError(t, errs[idx])
				}
			}
		})
	}
}

func (s *CacheTestSuite) TestMemoized() {
	t := s.T()

	// Arrange
	var (
		cached  memoized[int, int]
		calls   atomic.Int32
		workers sync.WaitGroup
	)

	results := make([]int, 64)

	// Act
	for idx := range results {
		workers.Go(func() {
			key := idx % 4
			results[idx] = cached.Do(key, func() int {
				calls.Add(1)

				return key * 2
			})
		})
	}

	workers.Wait()

	// Assert
	require.EqualValues(t, 4, calls.Load())

	for idx, result := range results {
		require.Equal(t, (idx%4)*2, result)
	}
}

func (s *CacheTestSuite) TestMemoizedErrors() {
	t := s.T()

	// Arrange
	var cached memoized[string, int]

	want := errors.New("lookup failed")
	calls := 0
	lookup := func() (int, error) {
		calls++

		return 7, want
	}

	// Act
	first, firstError := cached.DoE("missing", lookup)
	second, secondError := cached.DoE("missing", lookup)
	other, otherError := cached.DoE("other", lookup)

	// Assert
	require.Equal(t, 2, calls)
	require.Equal(t, 7, first)
	require.Equal(t, first, second)
	require.Equal(t, first, other)
	require.ErrorIs(t, firstError, want)
	require.ErrorIs(t, secondError, want)
	require.ErrorIs(t, otherError, want)
}

func TestCache(t *testing.T) {
	t.Parallel()

	suite.Run(t, &CacheTestSuite{})
}
