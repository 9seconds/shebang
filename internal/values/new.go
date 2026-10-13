package values

import (
	"github.com/spf13/cobra"
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
//nolint:ireturn,cyclop // Value interface dispatch needs one case per supported type.
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
	case "filepath":
		return NewFilepath(properties, subvalidators)
	case "base64":
		return NewBase64(properties, subvalidators)
	}

	return nil, ErrUnknownValueType
}
