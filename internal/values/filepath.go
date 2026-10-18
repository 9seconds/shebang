package values

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/world"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/spf13/cobra"
)

var filepathTypeTests = map[string]func(string) bool{
	"exists": func(input string) bool {
		_, err := world.World.Files.Stat(input)

		return err == nil
	},
	"absent": func(input string) bool {
		_, err := world.World.Files.Stat(input)

		return errors.Is(err, fs.ErrNotExist)
	},
	"regular": func(input string) bool {
		if info, err := world.World.Files.Stat(input); err == nil {
			return info.Mode().IsRegular()
		}

		return false
	},
	"directory": func(input string) bool {
		if info, err := world.World.Files.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeDir != 0
		}

		return false
	},
	"symlink": func(input string) bool {
		if info, err := world.World.Files.Lstat(input); err == nil {
			return info.Mode().Type()&fs.ModeSymlink != 0
		}

		return false
	},
	"pipe": func(input string) bool {
		if info, err := world.World.Files.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeNamedPipe != 0
		}

		return false
	},
	"socket": func(input string) bool {
		if info, err := world.World.Files.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeSocket != 0
		}

		return false
	},
	"device": func(input string) bool {
		if info, err := world.World.Files.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeDevice != 0
		}

		return false
	},
	"char-device": func(input string) bool {
		if info, err := world.World.Files.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeCharDevice != 0
		}

		return false
	},
}

type valueFilepath struct {
	baseValue[string]

	glob string
}

// NewFilepath constructs a filesystem path validator with optional constraints.
//
//nolint:ireturn // Filepath validators expose the shared Value interface.
func NewFilepath(
	properties map[string]any,
	subvalidators map[string][]any,
) (Value, error) {
	val := &valueFilepath{
		baseValue: newBase[string]("filepath", len(subvalidators)),
	}
	val.toValue = func(input string) (string, error) {
		if input == "" {
			return "", ErrFilepathNotSet
		}

		return filepath.Abs(input)
	}

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", k, err)
		}
	}

	return val, nil
}

// Complete bypasses baseValue's valid-input shortcut: a valid path can still be
// a partial filename or directory that should expand into completion candidates.
func (v *valueFilepath) Complete(input string) ([]string, cobra.ShellCompDirective) {
	return v.genCompletions(input)
}

func (v *valueFilepath) genCompletions(input string) ([]string, cobra.ShellCompDirective) {
	if v.glob != "" {
		return v.genCompletionsGlob(input)
	}

	return v.genCompletionsCwd(input)
}

func (v *valueFilepath) genCompletionsCwd(input string) ([]string, cobra.ShellCompDirective) {
	directory, prefix := filepath.Split(input)

	directoryToRead := directory
	if directory == "" {
		directoryToRead = "."
	}

	files, err := os.ReadDir(directoryToRead)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	paths := make([]string, 0, len(files))
	directive := cobra.ShellCompDirectiveNoFileComp

	for _, entry := range files {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		// we use '+' here instead to filepath.Join to preserve leading dot.
		// for example, if user has typed './report.json', then join makes
		// 'report.json' because of filepath.Clean. In all cases except of
		// shell completion this is fine, but we want to complete.
		fullPath := directory + name

		isDir := entry.IsDir()
		// DirEntry uses Lstat, so symlink here is not followed. We want to
		// follow, so let's resolve it on our own
		if !isDir && entry.Type()&os.ModeSymlink != 0 {
			if info, err := world.World.Files.Stat(fullPath); err == nil {
				isDir = info.IsDir()
			}
		}

		switch {
		case isDir:
			paths = append(paths, fullPath+string(os.PathSeparator))
			directive |= cobra.ShellCompDirectiveNoSpace
		case v.Validate(fullPath) == nil:
			paths = append(paths, fullPath)
		}
	}

	return paths, directive
}

func (v *valueFilepath) genCompletionsGlob(input string) ([]string, cobra.ShellCompDirective) {
	matches, err := doublestar.FilepathGlob(v.glob)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		absolute, err := filepath.Abs(match)
		if err != nil {
			continue
		}

		relative, err := filepath.Rel(cwd, absolute)

		switch {
		case
			// filepath.Rel fails when paths cannot be made relative (e.g., on
			// different Windows volumes or if a path is not lexical).
			err != nil,
			// A broad glob such as "**" can match the working directory itself;
			// skip "." so completion only suggests entries under cwd.
			relative == ".",
			// A glob such as "../**/*.json" or "/tmp/*.json" can match files
			// outside cwd, producing "../foo.json".
			strings.HasPrefix(relative, ".."+string(os.PathSeparator)):
			continue
		case strings.HasPrefix(input, "."+string(os.PathSeparator)):
			// filepath.Rel strips a leading "./" (returning "report.json"),
			// so restore "./" when the user is completing "./rep" -> "./report.json".
			relative = "." + string(os.PathSeparator) + relative
		}

		if !strings.HasPrefix(relative, input) {
			continue
		}

		if v.Validate(relative) == nil {
			paths = append(paths, relative)
		}
	}

	slices.Sort(paths)

	return slices.Compact(paths), cobra.ShellCompDirectiveNoFileComp
}

//nolint:cyclop // Dispatch each supported subvalidator to its configuration helper.
func (v *valueFilepath) addSubvalidator(name string, properties []any) error {
	switch name {
	case "glob":
		return v.addSubvalidatorGlob(properties)
	case "type": //nolint:goconst // Keep subvalidator names explicit in switch cases.
		return v.addSubvalidatorType(properties)
	case "permissions":
		return v.addSubvalidatorPermissions(properties)
	case "owner-name":
		return v.addSubvalidatorOwnerName(properties)
	case "owner-id":
		return v.addSubvalidatorOwnerID(properties)
	case "owner-me":
		return v.addSubvalidatorOwnerMe(properties)
	case "group-name":
		return v.addSubvalidatorGroupName(properties)
	case "group-id":
		return v.addSubvalidatorGroupID(properties)
	case "group-me":
		return v.addSubvalidatorGroupMe(properties)
	case "readable":
		return v.addSubvalidatorReadable(properties)
	case "writable":
		return v.addSubvalidatorWritable(properties)
	case "executable":
		return v.addSubvalidatorExecutable(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueFilepath) addSubvalidatorGlob(properties []any) error {
	expr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	if !doublestar.ValidatePathPattern(expr) {
		return ErrFilepathInvalidGlob
	}

	v.glob = expr

	pattern, err := filepath.Abs(expr)
	if err != nil {
		return err
	}

	v.setSubvalidator("glob", expr, func(input string) error {
		if !doublestar.PathMatchUnvalidated(pattern, input) {
			return fmt.Errorf("%w: %s", ErrFilepathGlobMismatch, expr)
		}

		return nil
	})

	return nil
}

func (v *valueFilepath) addSubvalidatorType(properties []any) error {
	types, err := utils.All[string](properties)
	if err != nil {
		return err
	}

	for _, typ := range types {
		if _, ok := filepathTypeTests[typ]; !ok {
			return fmt.Errorf("%w %s", ErrFilepathUnknownType, typ)
		}
	}

	slices.Sort(types)
	types = slices.Compact(types)

	v.setSubvalidator("type", strings.Join(types, ","), func(input string) error {
		for _, typ := range types {
			if filepathTypeTests[typ](input) {
				return nil
			}
		}

		return ErrFilepathNoSuitableType
	})

	return nil
}

func (v *valueFilepath) addSubvalidatorPermissions(properties []any) error {
	permissions, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if permissions < 0 || permissions > int64(fs.ModePerm) {
		return ErrFilepathIncorrectPermissions
	}

	perm := fs.FileMode(permissions)

	v.setSubvalidator("permissions", perm.String(), func(input string) error {
		info, err := world.World.Files.Stat(input)
		if err != nil {
			return err
		}

		if pm := info.Mode().Perm(); pm != perm {
			return fmt.Errorf("%w: expected %O, got %O", ErrFilepathIncorrectPermissions, perm, pm)
		}

		return nil
	})

	return nil
}

func (v *valueFilepath) addSubvalidatorOwnerName(properties []any) error {
	name, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	uid, err := world.World.Users.GetUser(name)
	if err != nil {
		return fmt.Errorf("cannot find a user: %w", err)
	}

	v.setSubvalidator("owner-name", name, v.makeOwnerValidator(uid, true))

	return nil
}

func (v *valueFilepath) addSubvalidatorOwnerID(properties []any) error {
	uidInt, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if uidInt < 0 || uidInt > math.MaxUint32 {
		return ErrFilepathIncorrectUID
	}

	uid := uint32(uidInt)

	v.setSubvalidator(
		"owner-id",
		strconv.FormatInt(uidInt, 10),
		v.makeOwnerValidator(uid, true),
	)

	return nil
}

func (v *valueFilepath) addSubvalidatorOwnerMe(properties []any) error {
	isMe, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	uid, err := world.World.Users.CurrentUser()
	if err != nil {
		return err
	}

	v.setSubvalidator(
		"owner-me",
		strconv.FormatBool(isMe),
		v.makeOwnerValidator(uid, isMe),
	)

	return nil
}

func (v *valueFilepath) addSubvalidatorGroupName(properties []any) error {
	name, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	gid, err := world.World.Users.GetGroup(name)
	if err != nil {
		return err
	}

	v.setSubvalidator(
		"group-name",
		name,
		v.makeGroupValidator([]uint32{gid}, true),
	)

	return nil
}

func (v *valueFilepath) addSubvalidatorGroupID(properties []any) error {
	gidInt, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if gidInt < 0 || gidInt > math.MaxUint32 {
		return ErrFilepathIncorrectGID
	}

	gid := uint32(gidInt)

	v.setSubvalidator(
		"group-id",
		strconv.FormatInt(gidInt, 10),
		v.makeGroupValidator([]uint32{gid}, true),
	)

	return nil
}

func (v *valueFilepath) addSubvalidatorGroupMe(properties []any) error {
	isMe, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	groups, err := world.World.Users.CurrentGroups()
	if err != nil {
		return err
	}

	v.setSubvalidator(
		"group-me",
		strconv.FormatBool(isMe),
		v.makeGroupValidator(groups, isMe),
	)

	return nil
}

func (v *valueFilepath) addSubvalidatorReadable(properties []any) error {
	isReadable, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("readable", strconv.FormatBool(isReadable), func(input string) error {
		err := world.World.Files.Readable(input)

		if err != nil && !errors.Is(err, fs.ErrPermission) {
			return err
		}

		if isReadable == (err == nil) {
			return nil
		}

		if isReadable {
			return ErrFilepathIsNotReadable
		}

		return ErrFilepathIsReadable
	})

	return nil
}

func (v *valueFilepath) addSubvalidatorWritable(properties []any) error {
	isWritable, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("writable", strconv.FormatBool(isWritable), func(input string) error {
		err := world.World.Files.Writable(input)

		if err != nil && !errors.Is(err, fs.ErrPermission) {
			return err
		}

		if isWritable == (err == nil) {
			return nil
		}

		if isWritable {
			return ErrFilepathIsNotWritable
		}

		return ErrFilepathIsWritable
	})

	return nil
}

func (v *valueFilepath) addSubvalidatorExecutable(properties []any) error {
	isExecutable, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("executable", strconv.FormatBool(isExecutable), func(input string) error {
		err := world.World.Files.Executable(input)

		if err != nil && !errors.Is(err, fs.ErrPermission) {
			return err
		}

		if isExecutable == (err == nil) {
			return nil
		}

		if isExecutable {
			return ErrFilepathIsNotExecutable
		}

		return ErrFilepathIsExecutable
	})

	return nil
}

func (v *valueFilepath) makeGroupValidator(groups []uint32, equal bool) func(string) error {
	return func(input string) error {
		info, err := world.World.Files.Stat(input)
		if err != nil {
			return err
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return ErrFilepathMissingMetadata
		}

		got := stat.Gid

		if slices.Contains(groups, got) == equal {
			return nil
		}

		if !equal {
			return fmt.Errorf("%w: owner group %d must not belong to %v",
				ErrFilepathIncorrectGID, got, groups)
		}

		return fmt.Errorf("%w: owner group %d, expected %v", ErrFilepathIncorrectGID, got, groups)
	}
}

func (v *valueFilepath) makeOwnerValidator(uid uint32, equal bool) func(string) error {
	return func(input string) error {
		info, err := world.World.Files.Stat(input)
		if err != nil {
			return err
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return ErrFilepathMissingMetadata
		}

		got := stat.Uid

		if (got == uid) == equal {
			return nil
		}

		if !equal {
			return fmt.Errorf("%w: owner %d must not equal %d", ErrFilepathIncorrectUID, got, uid)
		}

		return fmt.Errorf("%w: owner %d, expected %d", ErrFilepathIncorrectUID, got, uid)
	}
}
