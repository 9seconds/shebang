package values

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/spf13/cobra"
)

type valueSubnet struct {
	baseValue[netip.Prefix]

	overlaps            []netip.Prefix
	disableAutocomplete bool
}

// NewSubnet constructs a CIDR-prefix validator with family and overlap constraints.
//
//nolint:ireturn // Subnet validators expose the shared Value interface.
func NewSubnet(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueSubnet{
		baseValue: newBase[netip.Prefix]("subnet", len(subvalidators)),
	}
	val.toValue = netip.ParsePrefix
	val.toComplete = val.genCompletions

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("incorrect subvalidator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueSubnet) addSubvalidator(name string, properties []any) error {
	switch name {
	case "type": //nolint:goconst // Keep subvalidator names explicit in switch cases.
		return v.addSubvalidatorType(properties)
	case "overlaps":
		return v.addSubvalidatorOverlaps(properties)
	case "not-overlaps":
		return v.addSubvalidatorNotOverlaps(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueSubnet) addSubvalidatorType(properties []any) error {
	typ, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	subvalidator, ok := ipTypes[typ]
	if !ok {
		return NewErrAddressType(netip.Addr{}, typ)
	}

	v.setSubvalidator("type", typ, func(input netip.Prefix) error {
		if a := input.Addr(); !subvalidator(a) {
			return NewErrAddressType(a, typ)
		}

		return nil
	})

	return nil
}

func (v *valueSubnet) addSubvalidatorOverlaps(properties []any) error {
	prefixes, details, err := v.preparePrefixes(properties)
	if err != nil {
		return err
	}

	v.overlaps = prefixes

	v.setSubvalidator("overlaps", details, func(input netip.Prefix) error {
		for _, ntw := range prefixes {
			if ntw.Overlaps(input) {
				return nil
			}
		}

		return NewErrSubnetOverlap(input, prefixes, true)
	})

	return nil
}

func (v *valueSubnet) addSubvalidatorNotOverlaps(properties []any) error {
	prefixes, details, err := v.preparePrefixes(properties)
	if err != nil {
		return err
	}

	v.disableAutocomplete = true

	v.setSubvalidator("not-overlaps", details, func(input netip.Prefix) error {
		for _, ntw := range prefixes {
			if ntw.Overlaps(input) {
				return NewErrSubnetOverlap(input, []netip.Prefix{ntw}, false)
			}
		}

		return nil
	})

	return nil
}

func (v *valueSubnet) preparePrefixes(properties []any) ([]netip.Prefix, string, error) {
	if len(properties) == 0 {
		return nil, "", utils.ErrEmpty
	}

	texts, err := utils.All[string](properties)
	if err != nil {
		return nil, "", err
	}

	slices.Sort(texts)

	prefixes := make([]netip.Prefix, len(texts))

	for idx, txt := range texts {
		pfx, err := netip.ParsePrefix(txt)
		if err != nil {
			return nil, "", fmt.Errorf("incorrect prefix %s: %w", txt, err)
		}

		prefixes[idx] = pfx.Masked()
	}

	return prefixes, strings.Join(texts, ","), nil
}

func (v *valueSubnet) genCompletions(input string) ([]string, cobra.ShellCompDirective) {
	if v.disableAutocomplete {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	addresses := make([]string, 0, 2*len(v.overlaps))

	for _, pfx := range v.overlaps {
		if err := v.Validate(pfx.String()); err != nil {
			continue
		}

		addr := pfx.Addr()

		for _, v := range []string{addr.String(), addr.StringExpanded()} {
			v += "/"

			if strings.HasPrefix(v, input) {
				addresses = append(addresses, v)
			}
		}
	}

	directive := cobra.ShellCompDirectiveNoFileComp

	if len(addresses) > 0 {
		directive = cobra.ShellCompDirectiveNoSpace
	}

	return addresses, directive
}
