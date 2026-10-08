package validators

import "fmt"

func getOne[T any](values []any) (T, error) {
	var def T

	switch len(values) {
	case 0:
		return def, fmt.Errorf("1 value of %T must be defined", def)

	case 1:
		val, ok := values[0].(T)
		if !ok {
			return def, fmt.Errorf("expected %T parameter, but got %T", def, values[0])
		}
		return val, nil
	}

	return def, fmt.Errorf("expected 1 parameter of %T but got %d", def, len(values))
}
