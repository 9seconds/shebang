package values

import "uuid"

type valueUUID struct {
	baseValue[uuid.UUID]
}

// NewUUID constructs a UUID validator using the standard-library parser.
//
//nolint:ireturn // UUID validators expose the shared Value interface.
func NewUUID(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueUUID{
		baseValue: newBase[uuid.UUID]("uuid", 0),
	}
	val.toValue = uuid.Parse

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	if len(subvalidators) > 0 {
		return nil, ErrNoSubvalidators
	}

	return val, nil
}
