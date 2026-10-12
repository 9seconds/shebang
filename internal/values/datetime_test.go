package values_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type DatetimeTestSuite struct {
	suite.Suite
}

func (suite *DatetimeTestSuite) TestFormats() {
	for _, test := range []struct {
		name  string
		input string
	}{
		{
			name:  "iso8601",
			input: "2026-10-12T13:14:15.123+02:00",
		},
		{
			name:  "rfc822",
			input: "12 Oct 26 13:14 UTC",
		},
		{
			name:  "rfc822z",
			input: "12 Oct 26 13:14 +0200",
		},
		{
			name:  "rfc850",
			input: "Monday, 12-Oct-26 13:14:15 UTC",
		},
		{
			name:  "rfc1123",
			input: "Mon, 12 Oct 2026 13:14:15 UTC",
		},
		{
			name:  "rfc1123z",
			input: "Mon, 12 Oct 2026 13:14:15 +0200",
		},
		{
			name:  "rfc3339",
			input: "2026-10-12T13:14:15Z",
		},
		{
			name:  "rfc3339nano",
			input: "2026-10-12T13:14:15.123456789Z",
		},
		{
			name:  "Y-m-d",
			input: "2024-02-29",
		},
		{
			name:  "d.m.Y",
			input: "29.02.2024",
		},
		{
			name:  "H:M:S",
			input: "13:14:15",
		},
		{
			name:  "H:M",
			input: "13:14",
		},
		{
			name:  "M:S",
			input: "14:15",
		},
		{
			name:  "custom:2006/01/02 15:04",
			input: "2026/10/12 13:14",
		},
		{
			name:  "unix",
			input: "-0001",
		},
		{
			name:  "unix-ms",
			input: "+0001234",
		},
	} {
		suite.Run(test.name, func() {
			value, err := values.NewDatetime(map[string]any{"format": test.name}, nil)
			suite.Require().NoError(err)
			suite.Equal("datetime()", value.String())
			suite.Require().NoError(value.Validate(test.input))
			suite.Require().Error(value.Validate("привет"))

			choices, directive := value.Complete(test.input)
			suite.Equal([]cobra.Completion{test.input}, choices)
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

			choices, directive = value.Complete("")
			suite.Require().Len(choices, 1)
			suite.Require().NoError(value.Validate(choices[0]))
			suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func (suite *DatetimeTestSuite) TestDefaultFormat() {
	value, err := values.NewDatetime(nil, nil)
	suite.Require().NoError(err)
	suite.Require().NoError(value.Validate("2026-10-12T13:14:15Z"))
	suite.Require().NoError(value.Validate("2026-10-12"))
	suite.Require().Error(value.Validate("2026-02-30"))

	choices, directive := value.Complete("привет")
	suite.Empty(choices)
	suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
}

func (suite *DatetimeTestSuite) TestPropertyErrors() {
	value, err := values.NewDatetime(map[string]any{"unknown": true}, nil)
	suite.Require().ErrorIs(err, values.ErrUnknownProperty)
	suite.Require().ErrorContains(err, "cannot add the property unknown")
	suite.Nil(value)

	for _, property := range []any{true, int64(1), nil} {
		value, err = values.NewDatetime(map[string]any{"format": property}, nil)
		suite.Nil(value)

		var typeError *utils.ArgumentTypeError

		suite.Require().ErrorAs(err, &typeError)
		suite.Equal(*utils.NewArgumentTypeError("", property), *typeError)
	}

	value, err = values.NewDatetime(map[string]any{"format": "привет"}, nil)
	suite.Nil(value)

	var formatError *values.DatetimeFormatError

	suite.Require().ErrorAs(err, &formatError)
	suite.Equal(*values.NewDatetimeFormatError("привет"), *formatError)
	suite.Equal("unknown time format привет", formatError.Error())
}

func (suite *DatetimeTestSuite) TestSubvalidatorErrors() {
	for _, name := range []string{"before", "after", "date-before", "date-after"} {
		suite.Run(name, func() {
			for _, test := range []struct {
				name       string
				properties []any
				want       error
				typeError  bool
			}{
				{
					name: "missing argument",
					want: utils.ErrEmpty,
				},
				{
					name:       "extra argument",
					properties: []any{"1h", "2h"},
					want:       utils.ErrSingleArgumentExpected,
				},
				{
					name:       "wrong type",
					properties: []any{true},
					typeError:  true,
				},
				{
					name:       "invalid duration or date",
					properties: []any{"привет"},
				},
			} {
				suite.Run(test.name, func() {
					value, err := values.NewDatetime(nil, map[string][]any{name: test.properties})
					suite.Nil(value)
					suite.Require().ErrorContains(err, "cannot add validator "+name)

					if test.want != nil {
						suite.Require().ErrorIs(err, test.want)
					}

					if test.typeError {
						var typeError *utils.ArgumentTypeError

						suite.Require().ErrorAs(err, &typeError)
						suite.Equal("string", typeError.Expected)
					}
				})
			}
		})
	}

	value, err := values.NewDatetime(nil, map[string][]any{"unknown": {"1h"}})
	suite.Nil(value)
	suite.Require().ErrorIs(err, values.ErrUnknownSubvalidator)
}

func (suite *DatetimeTestSuite) TestAbsoluteBounds() {
	for _, format := range []string{"iso8601", "unix", "unix-ms", "custom:2006-01-02T15:04:05Z07:00"} {
		suite.Run(format, func() {
			bound := time.Date(2026, time.October, 12, 13, 14, 15, 0, time.UTC)
			formatTime := datetimeTestFormatter(format)

			for _, relation := range []string{"before", "after"} {
				suite.Run(relation, func() {
					name := "date-" + relation
					value, err := values.NewDatetime(map[string]any{"format": format},
						map[string][]any{name: {formatTime(bound)}})
					suite.Require().NoError(err)
					suite.Equal("datetime("+name+":"+formatTime(bound)+")", value.String())

					for _, offset := range []time.Duration{-time.Second, 0, time.Second} {
						input := bound.Add(offset)
						err = value.Validate(formatTime(input))

						if (relation == "before" && offset < 0) || (relation == "after" && offset > 0) {
							suite.Require().NoError(err)

							continue
						}

						var constraintError *values.DatetimeConstraintError

						suite.Require().ErrorAs(err, &constraintError)
						suite.True(input.Equal(constraintError.Value))
						suite.True(bound.Equal(constraintError.Bound))
						suite.Equal(relation, constraintError.Relation)
						suite.Equal(values.NewDatetimeConstraintError(input, bound, relation).Error(),
							constraintError.Error())
					}
				})
			}
		})
	}
}

func (suite *DatetimeTestSuite) TestRelativeBounds() {
	for _, relation := range []string{"before", "after"} {
		suite.Run(relation, func() {
			for _, duration := range []string{"1h", "-1h", "0s"} {
				suite.Run(duration, func() {
					started := time.Now()
					value, err := values.NewDatetime(nil, map[string][]any{relation: {duration}})
					suite.Require().NoError(err)

					finished := time.Now()
					shift, err := time.ParseDuration(duration)
					suite.Require().NoError(err)
					suite.Equal("datetime("+relation+":"+duration+")", value.String())

					past := started.Add(-48 * time.Hour)
					future := finished.Add(48 * time.Hour)
					accepted, rejected := past, future

					if relation == "after" {
						accepted, rejected = future, past
					}

					suite.Require().NoError(value.Validate(accepted.Format(time.RFC3339Nano)))

					var constraintError *values.DatetimeConstraintError

					suite.Require().ErrorAs(value.Validate(rejected.Format(time.RFC3339Nano)),
						&constraintError)
					suite.False(constraintError.Bound.Before(started.Add(shift)))
					suite.False(constraintError.Bound.After(finished.Add(shift)))
				})
			}
		})
	}
}

func (suite *DatetimeTestSuite) TestCombinedBoundsAndCompletion() {
	value, err := values.NewDatetime(map[string]any{"format": "unix"}, map[string][]any{
		"date-after":  {"-200"},
		"date-before": {"-100"},
	})
	suite.Require().NoError(err)
	suite.Require().NoError(value.Validate("-150"))
	suite.Require().Error(value.Validate("-200"))
	suite.Require().Error(value.Validate("-100"))
	suite.Equal("datetime(date-after:-200, date-before:-100)", value.String())

	choices, directive := value.Complete("-")
	suite.Equal([]cobra.Completion{"-100", "-200"}, choices)
	suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

	choices, directive = value.Complete("-1")
	suite.Equal([]cobra.Completion{"-100"}, choices)
	suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

	value, err = values.NewDatetime(map[string]any{"format": "unix-ms"}, map[string][]any{
		"date-before": {"-1001"},
		"date-after":  {"-1001"},
	})
	suite.Require().NoError(err)

	choices, directive = value.Complete("-")
	suite.Equal([]cobra.Completion{"-1001"}, choices)
	suite.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
}

func datetimeTestFormatter(format string) func(time.Time) string {
	return func(input time.Time) string {
		switch format {
		case "unix":
			return strconv.FormatInt(input.Unix(), 10)
		case "unix-ms":
			return strconv.FormatInt(input.UnixMilli(), 10)
		default:
			return input.Format(time.RFC3339)
		}
	}
}

func TestDatetime(t *testing.T) {
	t.Parallel()

	suite.Run(t, &DatetimeTestSuite{})
}
