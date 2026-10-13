package values

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/mcnijman/go-emailaddress"
)

type valueEmail struct {
	baseValue[*emailaddress.EmailAddress]
}

// NewEmail constructs an email validator with domain, local-part, and suffix constraints.
//
//nolint:ireturn // Email validators expose the shared Value interface.
func NewEmail(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueEmail{
		baseValue: newBase[*emailaddress.EmailAddress]("email", len(subvalidators)),
	}
	val.toValue = func(input string) (*emailaddress.EmailAddress, error) {
		if input == "" {
			return nil, ErrEmailEmpty
		}

		return emailaddress.Parse(input)
	}

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("cannot add subvalidator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueEmail) addSubvalidator(name string, properties []any) error {
	switch name {
	case "domain":
		return v.addSubvalidatorDomain(properties)
	case "domain-re":
		return v.addSubvalidatorDomainRe(properties)
	case "user-re":
		return v.addSubvalidatorUserRe(properties)
	case "icann":
		return v.addSubvalidatorICANN(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueEmail) addSubvalidatorDomain(properties []any) error {
	domain, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("domain", domain, func(input *emailaddress.EmailAddress) error {
		if input.Domain != domain {
			return ErrEmailDomain
		}

		return nil
	})

	return nil
}

func (v *valueEmail) addSubvalidatorDomainRe(properties []any) error {
	domainExpr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	domainRe, err := regexp.Compile(domainExpr)
	if err != nil {
		return err
	}

	v.setSubvalidator("domain-re", domainExpr, func(input *emailaddress.EmailAddress) error {
		if !domainRe.MatchString(input.Domain) {
			return ErrEmailDomain
		}

		return nil
	})

	return nil
}

func (v *valueEmail) addSubvalidatorUserRe(properties []any) error {
	userExpr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	userRe, err := regexp.Compile(userExpr)
	if err != nil {
		return err
	}

	v.setSubvalidator("user-re", userExpr, func(input *emailaddress.EmailAddress) error {
		if !userRe.MatchString(input.LocalPart) {
			return ErrEmailUser
		}

		return nil
	})

	return nil
}

func (v *valueEmail) addSubvalidatorICANN(properties []any) error {
	isICANN, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	validator := func(input *emailaddress.EmailAddress) error {
		if isICANN == (input.ValidateIcanSuffix() == nil) {
			return nil
		}

		if isICANN {
			return ErrEmailICANNRequired
		}

		return ErrEmailICANNExcluded
	}

	v.setSubvalidator("icann", strconv.FormatBool(isICANN), validator)

	return nil
}
