package values

import "errors"

// ErrFloatNonFinite indicates NaN or an infinite float value or bound.
var ErrFloatNonFinite = errors.New("float must be finite")
