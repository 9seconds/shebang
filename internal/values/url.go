package values

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/bmatcuk/doublestar/v4"
)

type valueURL struct {
	baseValue[*url.URL]
}

// NewURL constructs a URL-reference validator with component constraints.
//
//nolint:ireturn // URL validators expose the shared Value interface.
func NewURL(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueURL{
		baseValue: newBase[*url.URL]("url", len(subvalidators)),
	}
	val.toValue = url.Parse

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

//nolint:cyclop // Each supported subvalidator has one dispatch case.
func (v *valueURL) addSubvalidator(name string, properties []any) error {
	switch name {
	case "scheme":
		return v.addSubvalidatorScheme(properties)
	case "user":
		return v.addSubvalidatorUser(properties)
	case "user-re":
		return v.addSubvalidatorUserRe(properties)
	case "password":
		return v.addSubvalidatorPassword(properties)
	case "domain":
		return v.addSubvalidatorDomain(properties)
	case "domain-re":
		return v.addSubvalidatorDomainRe(properties)
	case "path-re":
		return v.addSubvalidatorPathRe(properties)
	case "path-glob":
		return v.addSubvalidatorPathGlob(properties)
	case "query":
		return v.addSubvalidatorQuery(properties)
	case "query-re":
		return v.addSubvalidatorQueryRe(properties)
	case "fragment-re":
		return v.addSubvalidatorFragmentRe(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueURL) addSubvalidatorScheme(properties []any) error {
	scheme, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("scheme", scheme, func(input *url.URL) error {
		if input.Scheme != scheme {
			return NewErrURLConstraint("scheme", "equal", scheme, input.Scheme)
		}

		return nil
	})

	return nil
}

func (v *valueURL) addSubvalidatorUser(properties []any) error {
	usr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("user", usr, func(input *url.URL) error {
		if name := input.User.Username(); name != usr {
			return NewErrURLConstraint("user", "equal", usr, name)
		}

		return nil
	})

	return nil
}

func (v *valueURL) addSubvalidatorUserRe(properties []any) error {
	usrExpr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	usrRe, err := regexp.Compile(usrExpr)
	if err != nil {
		return err
	}

	v.setSubvalidator("user-re", usrExpr, func(input *url.URL) error {
		if name := input.User.Username(); !usrRe.MatchString(name) {
			return NewErrURLConstraint("user", "match regex", usrExpr, name)
		}

		return nil
	})

	return nil
}

func (v *valueURL) addSubvalidatorPassword(properties []any) error {
	have, err := utils.One[bool](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("password", strconv.FormatBool(have), func(input *url.URL) error {
		if _, set := input.User.Password(); set == have {
			return nil
		}

		if have {
			return ErrURLPasswordRequired
		}

		return ErrURLPasswordExcluded
	})

	return nil
}

func (v *valueURL) addSubvalidatorDomain(properties []any) error {
	domain, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("domain", domain, func(input *url.URL) error {
		if got := input.Hostname(); got != domain {
			return NewErrURLConstraint("domain", "equal", domain, got)
		}

		return nil
	})

	return nil
}

func (v *valueURL) addSubvalidatorDomainRe(properties []any) error {
	domainExpr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	domainRe, err := regexp.Compile(domainExpr)
	if err != nil {
		return err
	}

	v.setSubvalidator("domain-re", domainExpr, func(input *url.URL) error {
		if name := input.Hostname(); !domainRe.MatchString(name) {
			return NewErrURLConstraint("domain", "match regex", domainExpr, name)
		}

		return nil
	})

	return nil
}

func (v *valueURL) addSubvalidatorPathRe(properties []any) error {
	pathExpr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	pathRe, err := regexp.Compile(pathExpr)
	if err != nil {
		return err
	}

	v.setSubvalidator("path-re", pathExpr, func(input *url.URL) error {
		if !pathRe.MatchString(input.Path) {
			return NewErrURLConstraint("path", "match regex", pathExpr, input.Path)
		}

		return nil
	})

	return nil
}

func (v *valueURL) addSubvalidatorPathGlob(properties []any) error {
	pathGlob, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	if !doublestar.ValidatePattern(pathGlob) {
		return ErrURLInvalidGlob
	}

	v.setSubvalidator("path-glob", pathGlob, func(input *url.URL) error {
		if !doublestar.MatchUnvalidated(pathGlob, input.Path) {
			return NewErrURLConstraint("path", "match glob", pathGlob, input.Path)
		}

		return nil
	})

	return nil
}

func (v *valueURL) addSubvalidatorQuery(properties []any) error {
	expected, details, err := v.prepareQueries(properties)
	if err != nil {
		return err
	}

	v.setSubvalidator("query", details, func(input *url.URL) error {
		values, err := url.ParseQuery(input.RawQuery)
		if err != nil {
			return err
		}

		for key, expectedValue := range expected {
			if _, ok := values[key]; !ok {
				return fmt.Errorf("%w: %s", ErrURLQueryMissingKey, key)
			}

			for _, val := range values[key] {
				if val != expectedValue {
					return NewErrURLConstraint("query "+key, "equal", expectedValue, val)
				}
			}
		}

		return nil
	})

	return nil
}

func (v *valueURL) addSubvalidatorQueryRe(properties []any) error {
	expected, details, err := v.prepareQueries(properties)
	if err != nil {
		return err
	}

	expectedRe := make(map[string]*regexp.Regexp, len(expected))

	for key, expression := range expected {
		queryRe, err := regexp.Compile(expression)
		if err != nil {
			return fmt.Errorf("incorrect regexp %s: %w", expression, err)
		}

		expectedRe[key] = queryRe
	}

	v.setSubvalidator("query-re", details, func(input *url.URL) error {
		values, err := url.ParseQuery(input.RawQuery)
		if err != nil {
			return err
		}

		for key, queryRe := range expectedRe {
			if _, ok := values[key]; !ok {
				return fmt.Errorf("%w: %s", ErrURLQueryMissingKey, key)
			}

			for _, val := range values[key] {
				if !queryRe.MatchString(val) {
					return NewErrURLConstraint("query "+key, "match regex", queryRe.String(), val)
				}
			}
		}

		return nil
	})

	return nil
}

func (v *valueURL) addSubvalidatorFragmentRe(properties []any) error {
	fragmentExpr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	fragmentRe, err := regexp.Compile(fragmentExpr)
	if err != nil {
		return err
	}

	v.setSubvalidator("fragment-re", fragmentExpr, func(input *url.URL) error {
		if !fragmentRe.MatchString(input.Fragment) {
			return NewErrURLConstraint("fragment", "match regex", fragmentExpr, input.Fragment)
		}

		return nil
	})

	return nil
}

func (v *valueURL) prepareQueries(properties []any) (map[string]string, string, error) {
	pairs, err := utils.All[string](properties)
	if err != nil {
		return nil, "", err
	}

	if len(pairs) == 0 {
		return nil, "", ErrURLQueryEmpty
	}

	if len(pairs)%2 == 1 {
		return nil, "", fmt.Errorf("%w: %s", ErrURLQueryMissingValue, pairs[len(pairs)-1])
	}

	values := make(map[string]string, len(pairs)/2)
	details := make([]string, 0, len(pairs)/2)

	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		value := pairs[i+1]

		if _, ok := values[key]; ok {
			return nil, "", fmt.Errorf("%w: %s", ErrURLQueryDuplicateKey, key)
		}

		values[key] = value
		details = append(details, key+"="+value)
	}

	return values, strings.Join(details, ","), nil
}
