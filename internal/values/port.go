package values

import (
	"fmt"
	"math"
	"strconv"

	"github.com/9seconds/shebang/internal/utils"
)

const (
	// https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers#Well-known_ports
	portSystemLast = 1023
	// https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers#Registered_ports
	portRegisteredLast = 49151
)

type valuePort struct {
	baseValue[uint16]
}

// NewPort constructs a decimal uint16 port validator with optional range
// classifiers.
//
//nolint:ireturn // Port validators expose the shared Value interface.
func NewPort(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valuePort{
		baseValue: newBase[uint16]("port", len(subvalidators)),
	}
	val.toValue = func(input string) (uint16, error) {
			port, err := strconv.ParseUint(input, 10, 16)
			if err != nil {
				return 0, err
			}

			return uint16(port), nil
	}

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valuePort) addSubvalidator(name string, properties []any) error {
	val, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	switch name {
	case "well-known":
		v.addRangeSubvalidator(
			"well-known",
			0,
			portSystemLast,
			val,
		)
	case "registered":
		v.addRangeSubvalidator(
			"registered",
			portSystemLast+1,
			portRegisteredLast,
			val,
		)
	case "ephemeral":
		v.addRangeSubvalidator(
			"ephemeral",
			portRegisteredLast+1,
			math.MaxUint16,
			val,
		)
	default:
		return ErrUnknownSubvalidator
	}

	return nil
}

func (v *valuePort) addRangeSubvalidator(name string, first, last uint16, expected bool) {
	v.setSubvalidator(name, "", func(input uint16) error {
		if expected == (first <= input && input <= last) {
			return nil
		}

		return NewPortRangeError(input, name, expected)
	})
}
