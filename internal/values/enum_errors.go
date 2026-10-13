package values

import "errors"

// ErrEnumUnknownChoice indicates a value outside the configured enum choices.
var ErrEnumUnknownChoice = errors.New("unknown choice")
