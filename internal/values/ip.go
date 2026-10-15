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
func NewIP(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueIP{
		baseValue:   newBase[netip.Addr]("ip", len(subvalidators)),
		classifiers: make(map[string]bool, len(ipClassifiers)),
	}
	val.toValue = netip.ParseAddr
	val.toComplete = val.genCompletions

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	for name, properties := range subvalidators {
		if err := val.addSubvalidator(name, properties); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", name, err)
		}
	}

	val.setSubvalidator("classifiers", "", func(input netip.Addr) error {
		for name, expected := range val.classifiers {
			if ipClassifiers[name](input) != expected {
				return NewErrIPClassifier(input, name, expected)
			}
		}

		return nil
	})

	return val, nil
}

func (v *valueIP) addSubvalidator(name string, properties []any) error {
	switch name {
	case "subnets":
		return v.addSubnets(properties)
	case "type": //nolint:goconst // Keep subvalidator names explicit in switch cases.
		return v.addType(properties)
	}

	if _, ok := ipClassifiers[name]; !ok {
		return ErrUnknownSubvalidator
	}

	expected, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	v.classifiers[name] = expected

	details := make([]string, 0, len(v.classifiers))
	for name := range v.classifiers {
		details = append(details, name)
	}

	slices.Sort(details)

	v.subvalidatorDetails["classifiers"] = strings.Join(details, ",")

	return nil
}

func (v *valueIP) addSubnets(properties []any) error {
	if len(properties) == 0 {
		return utils.ErrEmpty
	}

	strPrefixes, err := utils.All[string](properties)
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

	v.setSubvalidator("subnets", strings.Join(strPrefixes, ","), func(input netip.Addr) error {
		for _, prefix := range v.prefixes {
			if prefix.Contains(input) {
				return nil
			}
		}

		return NewErrIPSubnet(input, v.prefixes)
	})

	return nil
}

func (v *valueIP) addType(properties []any) error {
	name, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	subvalidator, ok := ipTypes[name]
	if !ok {
		return NewErrAddressType(netip.Addr{}, name)
	}

	v.setSubvalidator("type", name, func(input netip.Addr) error {
		if !subvalidator(input) {
			return NewErrAddressType(input, name)
		}

		return nil
	})

	return nil
}

func (v *valueIP) genCompletions(input string) ([]string, cobra.ShellCompDirective) {
	addresses := make([]string, 0, 2*len(v.prefixes))

	isOk := func(addr netip.Addr) bool {
		for _, subvalidator := range v.subvalidators {
			if err := subvalidator(addr); err != nil {
				return false
			}
		}

		return true
	}

	for _, prefix := range v.prefixes {
		addr := prefix.Addr()

		if isOk(addr) {
			addresses = append(addresses, addr.String(), addr.StringExpanded())
		}
	}

	candidates := make([]cobra.Completion, 0, len(addresses))

	for _, address := range addresses {
		// cut 1 character out of the ip address because we do not
		// want actually to finish with an own address of the prefix,
		// we want to complete up to that string
		address = address[:len(address)-1]

		if strings.HasPrefix(address, input) {
			candidates = append(candidates, address)
		}
	}

	directive := cobra.ShellCompDirectiveNoFileComp
	if len(candidates) > 0 {
		directive = cobra.ShellCompDirectiveNoSpace
	}

	return candidates, directive
}
