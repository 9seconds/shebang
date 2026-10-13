package values

import "net"

type valueMacAddr struct {
	baseValue[net.HardwareAddr]
}

// NewMacAddr constructs a hardware-address validator.
//
//nolint:ireturn // Hardware-address validators expose the shared Value interface.
func NewMacAddr(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueMacAddr{
		baseValue: newBase[net.HardwareAddr]("mac-addr", 0),
	}
	val.toValue = net.ParseMAC

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	if len(subvalidators) > 0 {
		return nil, ErrNoSubvalidators
	}

	return val, nil
}
