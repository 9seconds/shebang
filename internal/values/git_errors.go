package values

import "errors"

var (
	// ErrGitUnknownType indicates an unsupported reference category.
	ErrGitUnknownType = errors.New("unknown git ref type")

	// ErrGitEmptyTypes indicates an empty type constraint.
	ErrGitEmptyTypes = errors.New("at least 1 type must be defined")

	// ErrGitReferenceNotFound indicates no match in the selected categories.
	ErrGitReferenceNotFound = errors.New("cannot find such reference")
)
