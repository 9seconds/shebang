package values

import (
	"fmt"
	"net/netip"
	"slices"
)

// ErrIPClassifier describes an address with an unexpected classification.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrIPClassifier struct {
	Address    netip.Addr
	Classifier string
	Expected   bool
}

// NewErrIPClassifier constructs an error for a failed classification constraint.
func NewErrIPClassifier(
	address netip.Addr,
	classifier string,
	expected bool,
) *ErrIPClassifier {
	return &ErrIPClassifier{
		Address:    address,
		Classifier: classifier,
		Expected:   expected,
	}
}

// Error describes the required and actual classification.
func (e *ErrIPClassifier) Error() string {
	return fmt.Sprintf(
		"IP address %s: %s must be %t",
		e.Address,
		e.Classifier,
		e.Expected,
	)
}

// ErrIPSubnet describes an address outside all configured subnets.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrIPSubnet struct {
	Address netip.Addr
	Subnets []netip.Prefix
}

// NewErrIPSubnet constructs an error with an independent copy of the allowed subnets.
func NewErrIPSubnet(address netip.Addr, subnets []netip.Prefix) *ErrIPSubnet {
	return &ErrIPSubnet{
		Address: address,
		Subnets: slices.Clone(subnets),
	}
}

// Error describes the address that failed subnet membership.
func (e *ErrIPSubnet) Error() string {
	return fmt.Sprintf(
		"IP address %s does not belong to any configured subnet",
		e.Address,
	)
}
