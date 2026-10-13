package values

import (
	"encoding/json"
)

type valueJSON struct {
	baseValue[string]
}

// NewJSON constructs a schema-free validator for one complete JSON value.
//
//nolint:ireturn // JSON validators expose the shared Value interface.
func NewJSON(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueJSON{
		baseValue: newBase[string]("json", 0),
	}
	val.toValue = func(input string) (string, error) {
		if err := json.Unmarshal([]byte(input), &json.RawMessage{}); err != nil {
			return "", err
		}

		return input, nil
	}

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	if len(subvalidators) > 0 {
		return nil, ErrNoSubvalidators
	}

	return val, nil
}
