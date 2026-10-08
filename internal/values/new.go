package values

import (
	"errors"

	"github.com/spf13/cobra"
)

var (
	ErrUnknownValueType = errors.New("unknown type")
	ErrUnknownProperty = errors.New("unknown property")
)

type Value interface {
	Validate(string) error
	Complete(string) ([]cobra.Completion, cobra.ShellCompDirective)
	String() string
}

func New(valueType string, properties map[string][]any) (Value, error) {
	switch valueType {
	case "":
		return NewStr(nil)
	case "str":
		return NewStr(properties)
	}

	return nil, ErrUnknownValueType
}
