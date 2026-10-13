package values

import "errors"

var (
	// ErrEmailEmpty indicates an empty email address.
	ErrEmailEmpty = errors.New("email address cannot be empty")
	// ErrEmailDomain indicates a domain rejected by a configured constraint.
	ErrEmailDomain = errors.New("unsupported domain")
	// ErrEmailUser indicates a local part rejected by its regular expression.
	ErrEmailUser = errors.New("unsupported user")
	// ErrEmailICANNRequired indicates a suffix not classified as ICANN-managed.
	ErrEmailICANNRequired = errors.New("email domain must have an ICANN-managed public suffix")
	// ErrEmailICANNExcluded indicates an ICANN-managed suffix excluded by configuration.
	ErrEmailICANNExcluded = errors.New("email domain must not have an ICANN-managed public suffix")
)
