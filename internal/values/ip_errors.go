package values

import (
	"fmt"
	"net/netip"
	"slices"
)

// IPTypeError describes an unknown family constraint or an address of the wrong family.
type IPTypeError struct {
	Address netip.Addr
	Type    string
}

// NewIPTypeError constructs an error for a family constraint.
// An invalid address indicates an unknown configured type.
func NewIPTypeError(address netip.Addr, family string) *IPTypeError {
	return &IPTypeError{
		Address: address,
		Type:    family,
	}
}

// Error describes the unknown type or rejected address.
func (e *IPTypeError) Error() string {
	if !e.Address.IsValid() {
		return "unknown IP type " + e.Type
	}

	return fmt.Sprintf("IP address %s is not of type %s", e.Address, e.Type)
}

// IPClassifierError describes an address with an unexpected classification.
type IPClassifierError struct {
	Address    netip.Addr
	Classifier string
	Expected   bool
}

// NewIPClassifierError constructs an error for a failed classification constraint.
func NewIPClassifierError(
	address netip.Addr,
	classifier string,
	expected bool,
) *IPClassifierError {
	return &IPClassifierError{
		Address:    address,
		Classifier: classifier,
		Expected:   expected,
	}
}

// Error describes the required and actual classification.
func (e *IPClassifierError) Error() string {
	return fmt.Sprintf(
		"IP address %s: %s must be %t",
		e.Address,
		e.Classifier,
		e.Expected,
	)
}

// IPSubnetError describes an address outside all configured subnets.
type IPSubnetError struct {
	Address netip.Addr
	Subnets []netip.Prefix
}

// NewIPSubnetError constructs an error with an independent copy of the allowed subnets.
func NewIPSubnetError(address netip.Addr, subnets []netip.Prefix) *IPSubnetError {
	return &IPSubnetError{
		Address: address,
		Subnets: slices.Clone(subnets),
	}
}

// Error describes the address that failed subnet membership.
func (e *IPSubnetError) Error() string {
	return fmt.Sprintf(
		"IP address %s does not belong to any configured subnet",
		e.Address,
	)
}
