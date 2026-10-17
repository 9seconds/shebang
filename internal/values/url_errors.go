package values

import (
	"errors"
	"fmt"
)

var (
	// ErrURLPasswordRequired indicates missing password userinfo.
	ErrURLPasswordRequired = errors.New("password must be defined")

	// ErrURLPasswordExcluded indicates disallowed password userinfo.
	ErrURLPasswordExcluded = errors.New("password must not be defined")

	// ErrURLInvalidGlob indicates a malformed path glob.
	ErrURLInvalidGlob = errors.New("invalid glob pattern")

	// ErrURLQueryEmpty indicates missing query constraint pairs.
	ErrURLQueryEmpty = errors.New("no query pairs were defined")

	// ErrURLQueryMissingValue indicates a key without a configured value.
	ErrURLQueryMissingValue = errors.New("no value for query key")

	// ErrURLQueryDuplicateKey indicates repeated keys in a query constraint.
	ErrURLQueryDuplicateKey = errors.New("duplicate query parameter")

	// ErrURLQueryMissingKey indicates a required key absent from the URL.
	ErrURLQueryMissingKey = errors.New("query must define key")
)

// ErrURLConstraint describes a URL component rejected by a constraint.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrURLConstraint struct {
	Component string
	Operation string
	Expected  string
	Actual    string
}

// NewErrURLConstraint constructs a URL component constraint error.
func NewErrURLConstraint(component, operation, expected, actual string) *ErrURLConstraint {
	return &ErrURLConstraint{
		Component: component,
		Operation: operation,
		Expected:  expected,
		Actual:    actual,
	}
}

// Error describes the required component and its actual value.
func (e *ErrURLConstraint) Error() string {
	return fmt.Sprintf("URL %s must %s %q, got %q", e.Component, e.Operation, e.Expected, e.Actual)
}
