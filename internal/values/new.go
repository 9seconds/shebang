package values

import (
	"errors"

	"github.com/spf13/cobra"
)

var (
	// ErrUnknownValueType indicates an unsupported value type.
	ErrUnknownValueType = errors.New("unknown type")

	// ErrUnknownSubvalidator indicates an unsupported subvalidator.
	ErrUnknownSubvalidator = errors.New("unknown subvalidator")

	// ErrUnknownProperty indicates an unsupported value property.
	ErrUnknownProperty = errors.New("unknown property")

	// ErrNoProperties indicates that the value does not support properties.
	ErrNoProperties = errors.New("does not support properties")
)

// Value validates input, provides shell completions, and describes its subvalidators.
type Value interface {
	Validate(string) error
	Complete(string) ([]cobra.Completion, cobra.ShellCompDirective)
	String() string
}

// New constructs a validator for valueType. An empty type selects an
// unconstrained string validator; explicit types apply the supplied subvalidators.
//
//nolint:ireturn // Configured value types share the Value interface.
func New(
	valueType string,
	properties map[string]any,
	subvalidators map[string][]any,
) (Value, error) {
	switch valueType {
	case "":
		return NewStr(nil, nil)
	case "str":
		return NewStr(properties, subvalidators)
	case "int":
		return NewInt(properties, subvalidators)
	case "float":
		return NewFloat(properties, subvalidators)
	case "ip":
		return NewIP(properties, subvalidators)
	case "port":
		return NewPort(properties, subvalidators)
	case "enum":
		return NewEnum(properties, subvalidators)
	case "datetime":
		return NewDatetime(properties, subvalidators)
	}

	return nil, ErrUnknownValueType
}
