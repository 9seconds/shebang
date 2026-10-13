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
	"github.com/spf13/cobra"
)

var (
	// ErrFilepathNotSet indicates an empty path.
	ErrFilepathNotSet = errors.New("no path is defined")
	// ErrFilepathIncorrectPermissions indicates invalid or mismatched permission bits.
	ErrFilepathIncorrectPermissions = errors.New("incorrect permissions")
	// ErrFilepathIncorrectUID indicates an invalid or mismatched owner UID.
	ErrFilepathIncorrectUID = errors.New("incorrect uid")
	// ErrFilepathIncorrectGID indicates an invalid or mismatched group GID.
	ErrFilepathIncorrectGID = errors.New("incorrect gid")
	// ErrFilepathIsReadable indicates unexpectedly allowed read access.
	ErrFilepathIsReadable = errors.New("path is readable")
	// ErrFilepathIsNotReadable indicates denied read access.
	ErrFilepathIsNotReadable = errors.New("path is not readable")
	// ErrFilepathIsWritable indicates unexpectedly allowed write access.
	ErrFilepathIsWritable = errors.New("path is writable")
	// ErrFilepathIsNotWritable indicates denied write access.
	ErrFilepathIsNotWritable = errors.New("path is not writable")
	// ErrFilepathIsExecutable indicates unexpectedly allowed execute access.
	ErrFilepathIsExecutable = errors.New("path is executable")
	// ErrFilepathIsNotExecutable indicates denied execute access.
	ErrFilepathIsNotExecutable = errors.New("path is not executable")
	// ErrFilepathNoSuitableType indicates that no configured path type matches.
	ErrFilepathNoSuitableType = errors.New("path has no suitable type")
	// ErrFilepathOutsideRoot indicates a path outside the configured lexical root.
	ErrFilepathOutsideRoot = errors.New("path is outside root")
	// ErrFilepathUnknownType indicates an unsupported filesystem type.
	ErrFilepathUnknownType = errors.New("unknown type")
	// ErrFilepathUnsupportedExtension indicates an extension outside the configured set.
	ErrFilepathUnsupportedExtension = errors.New("unsupported file extension")
	// ErrFilepathMissingMetadata indicates unavailable Unix ownership metadata.
	ErrFilepathMissingMetadata = errors.New("path has no Unix ownership metadata")
)

var filepathTypeTests = map[string]func(string) bool{
	"exists": func(input string) bool {
		_, err := utils.World.Stat(input)

		return err == nil
	},
	"absent": func(input string) bool {
		_, err := utils.World.Stat(input)

		return errors.Is(err, fs.ErrNotExist)
	},
	"regular": func(input string) bool {
		if info, err := utils.World.Stat(input); err == nil {
			return info.Mode().IsRegular()
		}

		return false
	},
	"directory": func(input string) bool {
		if info, err := utils.World.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeDir != 0
		}

		return false
	},
	"symlink": func(input string) bool {
		if info, err := utils.World.Lstat(input); err == nil {
			return info.Mode().Type()&fs.ModeSymlink != 0
		}

		return false
	},
	"pipe": func(input string) bool {
		if info, err := utils.World.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeNamedPipe != 0
		}

		return false
	},
	"socket": func(input string) bool {
		if info, err := utils.World.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeSocket != 0
		}

		return false
	},
	"device": func(input string) bool {
		if info, err := utils.World.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeDevice != 0
		}

		return false
	},
	"char-device": func(input string) bool {
		if info, err := utils.World.Stat(input); err == nil {
			return info.Mode().Type()&fs.ModeCharDevice != 0
		}

		return false
	},
}

type valueFilepath struct {
	baseValue[string]
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

// Complete suggests matching files and directories for continued navigation.
func (v *valueFilepath) Complete(input string) ([]string, cobra.ShellCompDirective) {
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

		fullPath := filepath.Join(directory, name)

		if entry.IsDir() {
			paths = append(paths, fullPath+string(os.PathSeparator))
			directive |= cobra.ShellCompDirectiveNoSpace
		} else if v.Validate(fullPath) == nil {
			paths = append(paths, fullPath)
		}
	}

	return paths, directive
}

//nolint:cyclop // Dispatch each supported subvalidator to its configuration helper.
func (v *valueFilepath) addSubvalidator(name string, properties []any) error {
	switch name {
	case "root":
		return v.addRoot(properties)
	case "type":
		return v.addType(properties)
	case "file-extensions":
		return v.addFileExtensions(properties)
	case "permissions":
		return v.addPermissions(properties)
	case "owner-name":
		return v.addOwnerName(properties)
	case "owner-id":
		return v.addOwnerID(properties)
	case "owner-me":
		return v.addOwnerMe(properties)
	case "group-name":
		return v.addGroupName(properties)
	case "group-id":
		return v.addGroupID(properties)
	case "group-me":
		return v.addGroupMe(properties)
	case "readable":
		return v.addReadable(properties)
	case "writable":
		return v.addWritable(properties)
	case "executable":
		return v.addExecutable(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueFilepath) addRoot(properties []any) error {
	path, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	path, err = v.toValue(path)
	if err != nil {
		return err
	}

	v.setSubvalidator("root", path, func(input string) error {
		relPath, err := filepath.Rel(path, input)
		if err != nil {
			return err
		}

		if !filepath.IsLocal(relPath) {
			return fmt.Errorf("%w: path %s should belong to %s", ErrFilepathOutsideRoot, input, path)
		}

		return nil
	})

	return nil
}

func (v *valueFilepath) addType(properties []any) error {
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

func (v *valueFilepath) addFileExtensions(properties []any) error {
	exts, err := utils.All[string](properties)
	if err != nil {
		return err
	}

	for idx, val := range exts {
		if !strings.HasPrefix(val, ".") {
			exts[idx] = "." + val
		}
	}

	slices.Sort(exts)
	exts = slices.Compact(exts)

	v.setSubvalidator("file-extensions", strings.Join(exts, ","), func(input string) error {
		if ext := filepath.Ext(input); !slices.Contains(exts, ext) {
			return fmt.Errorf("%w: file extension %s is not supported", ErrFilepathUnsupportedExtension, ext)
		}

		return nil
	})

	return nil
}

func (v *valueFilepath) addPermissions(properties []any) error {
	permissions, err := utils.One[int64](properties)
	if err != nil {
		return err
	}

	if permissions < 0 || permissions > int64(fs.ModePerm) {
		return ErrFilepathIncorrectPermissions
	}

	perm := fs.FileMode(permissions)

	v.setSubvalidator("permissions", perm.String(), func(input string) error {
		info, err := utils.World.Stat(input)
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

func (v *valueFilepath) addOwnerName(properties []any) error {
	name, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	uid, err := utils.World.LookupUser(name)
	if err != nil {
		return fmt.Errorf("cannot find a user: %w", err)
	}

	v.setSubvalidator("owner-name", name, v.makeOwnerValidator(uid, true))

	return nil
}

func (v *valueFilepath) addOwnerID(properties []any) error {
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

func (v *valueFilepath) addOwnerMe(properties []any) error {
	isMe, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	uid, err := utils.World.CurrentUser()
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

func (v *valueFilepath) addGroupName(properties []any) error {
	name, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	gid, err := utils.World.LookupGroup(name)
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

func (v *valueFilepath) addGroupID(properties []any) error {
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

func (v *valueFilepath) addGroupMe(properties []any) error {
	isMe, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	groups, err := utils.World.CurrentGroups()
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

func (v *valueFilepath) addReadable(properties []any) error {
	isReadable, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("readable", strconv.FormatBool(isReadable), func(input string) error {
		err := utils.World.IsReadable(input)

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

func (v *valueFilepath) addWritable(properties []any) error {
	isWritable, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("writable", strconv.FormatBool(isWritable), func(input string) error {
		err := utils.World.IsWritable(input)

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

func (v *valueFilepath) addExecutable(properties []any) error {
	isExecutable, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("executable", strconv.FormatBool(isExecutable), func(input string) error {
		err := utils.World.IsExecutable(input)

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
		info, err := utils.World.Stat(input)
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
		info, err := utils.World.Stat(input)
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
