package values

import (
	"errors"

	"github.com/spf13/cobra"
)

var (
	// ErrUnknownValueType indicates an unsupported value type.
	ErrUnknownValueType = errors.New("unknown type")
	// ErrUnknownProperty indicates an unsupported validation property.
	ErrUnknownProperty = errors.New("unknown property")
)

// Value validates input, provides shell completions, and describes its subvalidators.
type Value interface {
	Validate(string) error
	Complete(string) ([]cobra.Completion, cobra.ShellCompDirective)
	String() string
}

// New constructs a validator for valueType. An empty type selects an
// unconstrained string validator; explicit types apply the supplied properties.
//
//nolint:ireturn // Configured value types share the Value interface.
func New(valueType string, properties map[string][]any) (Value, error) {
	switch valueType {
	case "":
		return NewStr(nil)
	case "str":
		return NewStr(properties)
	case "int":
		return NewInt(properties)
	case "float":
		return NewFloat(properties)
	case "ip":
		return NewIP(properties)
	case "port":
		return NewPort(properties)
	}

	return nil, ErrUnknownValueType
}
