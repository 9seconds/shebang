package values

import (
	"github.com/spf13/cobra"
)

var newFactories = map[string]func(map[string]any, map[string][]any) (Value, error){
	"str":      NewStr,
	"int":      NewInt,
	"float":    NewFloat,
	"ip":       NewIP,
	"port":     NewPort,
	"enum":     NewEnum,
	"datetime": NewDatetime,
	"filepath": NewFilepath,
	"base64":   NewBase64,
	"uuid":     NewUUID,
	"mac-addr": NewMacAddr,
	"email":    NewEmail,
	"json":     NewJSON,
	"semver":   NewSemver,
}

// Value validates input, provides shell completions, and describes its subvalidators.
type Value interface {
	Validate(string) error
	Complete(string) ([]cobra.Completion, cobra.ShellCompDirective)
	String() string
}

// New constructs a validator for valueType. An empty type selects an
// unconstrained string validator; explicit types apply the supplied subvalidators.
//
//nolint:ireturn // Value interface is required here
func New(
	valueType string,
	properties map[string]any,
	subvalidators map[string][]any,
) (Value, error) {
	if valueType == "" {
		return NewStr(nil, nil)
	}

	if create, ok := newFactories[valueType]; ok {
		return create(properties, subvalidators)
	}

	return nil, ErrUnknownValueType
}
