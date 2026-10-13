package values

import "errors"

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
