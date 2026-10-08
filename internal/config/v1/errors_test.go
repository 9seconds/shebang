package v1_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/9seconds/shebang/internal/cli"
	v1 "github.com/9seconds/shebang/internal/config/v1"
	"github.com/stretchr/testify/suite"
)

type ArgumentCountErrorTestSuite struct {
	suite.Suite
}

func (suite *ArgumentCountErrorTestSuite) TestArgumentCountErrors() {
	runner, err := os.Executable()
	suite.Require().NoError(err)

	for _, test := range []struct {
		name    string
		doc     string
		args    []string
		want    v1.ArgumentCountError
		message string
	}{
		{
			name: "missing fixed and required variadic arguments",
			doc:  "arg \"source\"\nvararg \"items\" { min-count 2; }\narg \"dest\"\n",
			args: []string{"привет", "привет", "привет"},
			want: v1.ArgumentCountError{
				Kind:     v1.ArgumentCountMinimum,
				Expected: 4,
				Actual:   3,
			},
			message: "there must be at least 4 arguments, got 3",
		},
		{
			name: "unexpected variadic arguments exclude fixed arguments",
			doc:  "arg \"source\"\n",
			args: []string{"привет", "привет", "привет"},
			want: v1.ArgumentCountError{
				Kind:     v1.ArgumentCountUnexpectedVarArgs,
				Expected: 0,
				Actual:   2,
			},
			message: "expected 0 variadic arguments, got 2",
		},
		{
			name: "maximum includes required variadic items only",
			doc:  "arg \"source\"\nvararg \"items\" {\nmin-count 1\nmax-count 2\n}\narg \"dest\"\n",
			args: []string{"привет", "привет", "привет", "привет", "привет"},
			want: v1.ArgumentCountError{
				Kind:     v1.ArgumentCountMaximumVarArgs,
				Expected: 2,
				Actual:   3,
			},
			message: "there must be at most 2 variadic arguments, got 3",
		},
	} {
		suite.Run(test.name, func() {
			constructed := v1.NewArgumentCountError(test.want.Kind, test.want.Expected, test.want.Actual)
			suite.Require().NotNil(constructed)
			suite.Equal(test.want, *constructed)
			suite.Equal(test.message, constructed.Error())

			conf, err := v1.Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)

			conf.Argv = []string{runner}
			cmd := cli.NewCommand("script", nil)
			suite.Require().NoError(conf.Configure(cmd))

			err = cmd.Cmd.Args(&cmd.Cmd, test.args)
			suite.Require().EqualError(err, test.message)

			var countError *v1.ArgumentCountError

			suite.Require().ErrorAs(err, &countError)
			suite.Equal(test.want, *countError)
		})
	}
}

func (suite *ArgumentCountErrorTestSuite) TestVarArgBoundsError() {
	for _, test := range []struct {
		name    string
		minimum int64
		maximum int64
		message string
	}{
		{
			name:    "positive bounds",
			minimum: 3,
			maximum: 2,
			message: "min-count 3 is greater than max-count 2",
		},
		{
			name:    "zero maximum",
			minimum: 1,
			maximum: 0,
			message: "min-count 1 is greater than max-count 0",
		},
	} {
		suite.Run(test.name, func() {
			constructed := v1.NewVarArgBoundsError(test.minimum, test.maximum)
			suite.Require().NotNil(constructed)
			suite.Equal(test.minimum, constructed.Min)
			suite.Equal(test.maximum, constructed.Max)
			suite.Equal(test.message, constructed.Error())

			doc := fmt.Sprintf("vararg \"items\" {\nmin-count %d\nmax-count %d\n}\n",
				test.minimum, test.maximum)
			conf, err := v1.Parse(strings.NewReader(doc))
			suite.Require().ErrorContains(err, "cannot process node vararg:")
			suite.Require().ErrorContains(err, test.message)
			suite.Nil(conf)

			var boundsError *v1.VarArgBoundsError

			suite.Require().ErrorAs(err, &boundsError)
			suite.Equal(*constructed, *boundsError)
		})
	}
}

func TestArgumentCountError(t *testing.T) {
	t.Parallel()

	suite.Run(t, &ArgumentCountErrorTestSuite{})
}
