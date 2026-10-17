package world

import (
	"fmt"
	"os/user"
	"strconv"

	"github.com/9seconds/shebang/internal/log"
)

// Users caches user/group lookups and the current user's supplementary groups.
type Users struct {
	users  memoized[string, uint32]
	groups memoized[string, uint32]

	currentGroups funcOnce[[]uint32]
}

// CurrentUser returns the current user's UID.
func (u *Users) CurrentUser() (uint32, error) {
	return u.GetUser("")
}

// CurrentGroups returns cached group IDs, which callers must treat as read-only.
func (u *Users) CurrentGroups() ([]uint32, error) {
	return u.currentGroups.DoE(func() ([]uint32, error) {
		usr, err := user.Current()
		if err != nil {
			log.Printf("Users: cannot determine current user for group lookup: %s", err)

			return nil, fmt.Errorf("cannot get current user: %w", err)
		}

		groups, err := usr.GroupIds()
		if err != nil {
			log.Printf("Users: cannot list groups for user %q: %s", usr.Username, err)

			return nil, fmt.Errorf("cannot get current group ids: %w", err)
		}

		gids := make([]uint32, len(groups))

		for idx, sGid := range groups {
			gids[idx] = u.parseID(sGid)
		}

		log.Printf("Users: user %q belongs to group IDs %v", usr.Username, gids)

		return gids, nil
	})
}

// GetUser returns the UID for name; an empty name selects the current user.
func (u *Users) GetUser(name string) (uint32, error) {
	return u.users.DoE(name, func() (uint32, error) {
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
			log.Printf("Users: user lookup for %q failed (empty selects current user): %s", name, err)

			return 0, fmt.Errorf("cannot lookup %s user: %w", name, err)
		}

		uid := u.parseID(usr.Uid)
		log.Printf("Users: resolved user %q to UID %d", usr.Username, uid)

		return uid, nil
	})
}

// GetGroup returns the GID for a group name.
func (u *Users) GetGroup(name string) (uint32, error) {
	return u.groups.DoE(name, func() (uint32, error) {
		grp, err := user.LookupGroup(name)
		if err != nil {
			log.Printf("Users: group lookup for %q failed: %s", name, err)

			return 0, err
		}

		gid := u.parseID(grp.Gid)
		log.Printf("Users: resolved group %q to GID %d", name, gid)

		return gid, nil
	})
}

func (u *Users) parseID(value string) uint32 {
	val, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint32(val)
}
