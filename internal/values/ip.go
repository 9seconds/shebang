package values

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/spf13/cobra"
)

var ipTypes = map[string]func(netip.Addr) bool{
	"v4":       netip.Addr.Is4,
	"v6":       netip.Addr.Is6,
	"v4-in-v6": netip.Addr.Is4In6,
	"v6-only":  func(addr netip.Addr) bool { return addr.Is6() && !addr.Is4In6() },
}

var ipClassifiers = map[string]func(netip.Addr) bool{
	"global-unicast":      netip.Addr.IsGlobalUnicast,
	"iflocal-multicast":   netip.Addr.IsInterfaceLocalMulticast,
	"linklocal-unicast":   netip.Addr.IsLinkLocalUnicast,
	"linklocal-multicast": netip.Addr.IsLinkLocalMulticast,
	"loopback":            netip.Addr.IsLoopback,
	"multicast":           netip.Addr.IsMulticast,
	"private":             netip.Addr.IsPrivate,
	"unspecified":         netip.Addr.IsUnspecified,
}

type valueIP struct {
	baseValue[netip.Addr]

	prefixes    []netip.Prefix
	classifiers map[string]bool
}

// NewIP constructs an IP address validator with optional family, classifier,
// and subnet constraints. Completion suggests configured subnet base addresses.
//
//nolint:ireturn // IP validators expose the shared Value interface.
func NewIP(properties map[string][]any) (Value, error) {
	val := &valueIP{
		validatorType: "ip",
		checks:        make(map[string]func(netip.Addr) error, len(properties)),
		classifiers:   make(map[string]bool, len(ipClassifiers)),
		prepare:       netip.ParseAddr,
	}
	val.complete = val.completeAddresses

	for name, arguments := range properties {
		if err := val.addCheck(name, arguments); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", name, err)
		}
	}

	val.checks["classifiers"] = func(addr netip.Addr) error {
		for name, expected := range val.classifiers {
			if ipClassifiers[name](addr) != expected {
				return NewIPClassifierError(addr, name, expected)
			}
		}

		return nil
	}

	return val, nil
}

func (v *valueIP) addCheck(name string, arguments []any) error {
	switch name {
	case "subnets":
		return v.addSubnets(arguments)
	case "type":
		return v.addType(arguments)
	}

	if _, ok := ipClassifiers[name]; !ok {
		return ErrUnknownProperty
	}

	expected, err := utils.One[bool](arguments)
	if err != nil {
		return err
	}

	v.classifiers[name] = expected

	return nil
}

func (v *valueIP) addSubnets(arguments []any) error {
	if len(arguments) == 0 {
		return utils.ErrEmpty
	}

	strPrefixes, err := utils.All[string](arguments)
	if err != nil {
		return err
	}

	v.prefixes = make([]netip.Prefix, len(strPrefixes))

	for idx, text := range strPrefixes {
		prefix, err := netip.ParsePrefix(text)
		if err != nil {
			return fmt.Errorf("incorrect prefix %s: %w", text, err)
		}

		v.prefixes[idx] = prefix.Masked()
	}

	name := "subnets:" + strings.Join(strPrefixes, ",")

	v.checks[name] = func(addr netip.Addr) error {
		for _, prefix := range v.prefixes {
			if prefix.Contains(addr) {
				return nil
			}
		}

		return NewIPSubnetError(addr, v.prefixes)
	}

	return nil
}

func (v *valueIP) addType(arguments []any) error {
	name, err := utils.One[string](arguments)
	if err != nil {
		return err
	}

	check, ok := ipTypes[name]
	if !ok {
		return NewIPTypeError(netip.Addr{}, name)
	}

	v.checks["type:"+name] = func(addr netip.Addr) error {
		if !check(addr) {
			return NewIPTypeError(addr, name)
		}

		return nil
	}

	return nil
}

func (v *valueIP) completeAddresses(input string) ([]cobra.Completion, cobra.ShellCompDirective) {
	addresses := make(map[string]struct{}, 2*len(v.prefixes))

	isOk := func(addr netip.Addr) bool {
		for _, check := range v.checks {
			if err := check(addr); err != nil {
				return false
			}
		}

		return true
	}

	for _, prefix := range v.prefixes {
		addr := prefix.Addr()

		if isOk(addr) {
			addresses[addr.String()] = struct{}{}
			addresses[addr.StringExpanded()] = struct{}{}
		}
	}

	candidates := make([]cobra.Completion, 0, len(addresses))

	for address := range addresses {
		if strings.HasPrefix(address, input) {
			candidates = append(candidates, address)
		}
	}

	slices.Sort(candidates)

	return candidates, cobra.ShellCompDirectiveNoFileComp
}
