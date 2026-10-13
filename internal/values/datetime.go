package values

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/goschtalt/approx"
	"github.com/relvacode/iso8601"
	"github.com/spf13/cobra"
)

const datetimeDefaultFormat = "iso8601"

var datetimeLayouts = map[string]string{
	"rfc822":      time.RFC822,
	"rfc822z":     time.RFC822Z,
	"rfc850":      time.RFC850,
	"rfc1123":     time.RFC1123,
	"rfc1123z":    time.RFC1123Z,
	"rfc3339":     time.RFC3339,
	"rfc3339nano": time.RFC3339Nano,
	"Y-m-d":       "2006-01-02",
	"d.m.Y":       "02.01.2006",
	"H:M:S":       "15:04:05",
	"H:M":         "15:04",
	"M:S":         "04:05",
}

type valueDatetime struct {
	baseValue[time.Time]

	formatValue func(time.Time) string
	knownDates  map[string]time.Time
}

// NewDatetime constructs a datetime validator with configurable parsing and time bounds.
//
//nolint:ireturn // Datetime validators expose the shared Value interface.
func NewDatetime(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueDatetime{
		baseValue:  newBase[time.Time]("datetime", len(subvalidators)),
		knownDates: make(map[string]time.Time, len(subvalidators)),
	}
	val.toComplete = val.genCompletions

	if err := val.addPropFormat(datetimeDefaultFormat); err != nil {
		panic(err)
	}

	for name, prop := range properties {
		if err := val.addProp(name, prop); err != nil {
			return nil, fmt.Errorf("cannot add the property %s: %w", name, err)
		}
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("cannot add validator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueDatetime) addProp(name string, value any) error {
	if name == "format" {
		return v.addPropFormat(value)
	}

	return ErrUnknownProperty
}

//nolint:funlen // splitting it actually make it even worse
func (v *valueDatetime) addPropFormat(value any) error {
	strValue, ok := value.(string)
	if !ok {
		return utils.NewArgumentTypeError("", value)
	}

	if layout, ok := datetimeLayouts[strValue]; ok {
		v.toValue = func(input string) (time.Time, error) {
			return time.Parse(layout, input)
		}
		v.formatValue = func(input time.Time) string {
			return input.Format(layout)
		}

		return nil
	}

	if layout, ok := strings.CutPrefix(strValue, "custom:"); ok {
		v.toValue = func(input string) (time.Time, error) {
			return time.Parse(layout, input)
		}
		v.formatValue = func(input time.Time) string {
			return input.Format(layout)
		}

		return nil
	}

	switch strValue {
	// https://en.wikipedia.org/wiki/ISO_8601
	case "iso8601":
		v.toValue = iso8601.ParseString
		v.formatValue = func(input time.Time) string {
			// RFC3339Nano is allowed ISO8601 format
			return input.Format(time.RFC3339Nano)
		}

	// time in seconds from epoch
	case "unix":
		v.toValue = func(input string) (time.Time, error) {
			val, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				return time.Time{}, err
			}

			return time.Unix(val, 0), nil
		}
		v.formatValue = func(input time.Time) string {
			return strconv.FormatInt(input.Unix(), 10)
		}

	// time in MILLI-seconds from epoch
	case "unix-ms":
		v.toValue = func(input string) (time.Time, error) {
			val, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				return time.Time{}, err
			}

			return time.UnixMilli(val), nil
		}
		v.formatValue = func(input time.Time) string {
			return strconv.FormatInt(input.UnixMilli(), 10)
		}

	default:
		return NewErrDatetimeFormat(strValue)
	}

	return nil
}

func (v *valueDatetime) addSubvalidator(name string, properties []any) error {
	switch name {
	case "before":
		return v.addSubvalidatorBefore(properties)
	case "after":
		return v.addSubvalidatorAfter(properties)
	case "date-before":
		return v.addSubvalidatorDateBefore(properties)
	case "date-after":
		return v.addSubvalidatorDateAfter(properties)
	}

	return ErrUnknownSubvalidator
}

func (v *valueDatetime) addSubvalidatorBefore(properties []any) error {
	durationStr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	duration, err := approx.ParseDuration(durationStr)
	if err != nil {
		return err
	}

	cutOff := time.Now().Add(duration)
	v.setSubvalidator("before", durationStr, v.makeValidatorBefore(cutOff))
	v.knownDates["before"] = cutOff

	return nil
}

func (v *valueDatetime) addSubvalidatorAfter(properties []any) error {
	durationStr, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	duration, err := approx.ParseDuration(durationStr)
	if err != nil {
		return err
	}

	cutOff := time.Now().Add(duration)
	v.setSubvalidator("after", durationStr, v.makeValidatorAfter(cutOff))
	v.knownDates["after"] = cutOff

	return nil
}

func (v *valueDatetime) addSubvalidatorDateBefore(properties []any) error {
	timestamp, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	datetime, err := v.toValue(timestamp)
	if err != nil {
		return err
	}

	v.setSubvalidator("date-before", timestamp, v.makeValidatorBefore(datetime))
	v.knownDates["date-before"] = datetime

	return nil
}

func (v *valueDatetime) addSubvalidatorDateAfter(properties []any) error {
	timestamp, err := utils.One[string](properties)
	if err != nil {
		return err
	}

	datetime, err := v.toValue(timestamp)
	if err != nil {
		return err
	}

	v.setSubvalidator("date-after", timestamp, v.makeValidatorAfter(datetime))
	v.knownDates["date-after"] = datetime

	return nil
}

func (v *valueDatetime) makeValidatorAfter(datetime time.Time) func(time.Time) error {
	return func(input time.Time) error {
		if !input.After(datetime) {
			return NewErrDatetimeConstraint(input, datetime, "after")
		}

		return nil
	}
}

func (v *valueDatetime) makeValidatorBefore(datetime time.Time) func(time.Time) error {
	return func(input time.Time) error {
		if !input.Before(datetime) {
			return NewErrDatetimeConstraint(input, datetime, "before")
		}

		return nil
	}
}

func (v *valueDatetime) genCompletions(input string) ([]string, cobra.ShellCompDirective) {
	choices := make([]string, 0, 1+len(v.knownDates))

	if val := v.formatValue(time.Now()); strings.HasPrefix(val, input) {
		choices = append(choices, val)
	}

	for _, tme := range v.knownDates {
		if val := v.formatValue(tme); strings.HasPrefix(val, input) {
			choices = append(choices, val)
		}
	}

	return choices, cobra.ShellCompDirectiveNoFileComp
}
