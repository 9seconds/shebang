package world

import (
	"os/user"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UsersTestSuite struct {
	suite.Suite
}

func (s *UsersTestSuite) TestCurrent() {
	t := s.T()

	// Arrange
	var users Users

	current, err := user.Current()
	require.NoError(t, err)

	wantUID, err := strconv.ParseUint(current.Uid, 10, 32)
	require.NoError(t, err)

	wantGroups, err := current.GroupIds()
	require.NoError(t, err)

	group, err := user.LookupGroupId(current.Gid)
	require.NoError(t, err)

	wantGID, err := strconv.ParseUint(group.Gid, 10, 32)
	require.NoError(t, err)

	// Act
	uid, uidError := users.CurrentUser()
	namedUID, namedError := users.GetUser(current.Username)
	emptyUID, emptyError := users.GetUser("")
	gid, groupError := users.GetGroup(group.Name)
	groups, groupsError := users.CurrentGroups()

	// Assert
	require.NoError(t, uidError)
	require.Equal(t, uint32(wantUID), uid)

	require.NoError(t, namedError)
	require.Equal(t, uid, namedUID)

	require.NoError(t, emptyError)
	require.Equal(t, uid, emptyUID)

	require.NoError(t, groupError)
	require.Equal(t, uint32(wantGID), gid)

	require.NoError(t, groupsError)
	require.Len(t, groups, len(wantGroups))

	for _, text := range wantGroups {
		parsed, err := strconv.ParseUint(text, 10, 32)
		require.NoError(t, err)
		require.True(t, slices.Contains(groups, uint32(parsed)))
	}
}

func (s *UsersTestSuite) TestMissing() {
	t := s.T()

	// Arrange
	var users Users

	name := "shebang-no-such-identity-7b3e19"

	for _, lookup := range []func(string) (uint32, error){
		users.GetUser,
		users.GetGroup,
	} {
		// Act
		value, err := lookup(name)
		cached, cachedError := lookup(name)

		// Assert
		require.Zero(t, value)
		require.Error(t, err)
		require.Equal(t, value, cached)
		require.Equal(t, err, cachedError)
	}
}

func (s *UsersTestSuite) TestConcurrent() {
	t := s.T()

	// Arrange
	var (
		users   Users
		workers sync.WaitGroup
	)

	results := make([]uint32, 32)
	errs := make([]error, len(results))

	// Act
	for idx := range results {
		workers.Go(func() {
			results[idx], errs[idx] = users.CurrentUser()
			_, _ = users.CurrentGroups()
		})
	}

	workers.Wait()

	// Assert
	for idx, result := range results {
		require.NoError(t, errs[idx])
		require.Equal(t, results[0], result)
	}
}

func TestUsers(t *testing.T) {
	t.Parallel()

	suite.Run(t, &UsersTestSuite{})
}
