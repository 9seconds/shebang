package values

import (
	"fmt"
	"net/netip"
	"slices"
)

// ErrSubnetOverlap describes a failed required or excluded overlap constraint.
//
//nolint:errname // Project convention requires Err<Stem> names for all errors.
type ErrSubnetOverlap struct {
	Subnet   netip.Prefix
	Prefixes []netip.Prefix
	Expected bool
}

// NewErrSubnetOverlap constructs an overlap error with an independent prefix slice.
func NewErrSubnetOverlap(
	subnet netip.Prefix,
	prefixes []netip.Prefix,
	expected bool,
) *ErrSubnetOverlap {
	return &ErrSubnetOverlap{
		Subnet:   subnet,
		Prefixes: slices.Clone(prefixes),
		Expected: expected,
	}
}

// Error describes the subnet and its failed overlap requirement.
func (e *ErrSubnetOverlap) Error() string {
	if e.Expected {
		return fmt.Sprintf("subnet %s must overlap with at least one subnet: %v", e.Subnet, e.Prefixes)
	}

	return fmt.Sprintf("subnet %s must not overlap subnets: %v", e.Subnet, e.Prefixes)
}
