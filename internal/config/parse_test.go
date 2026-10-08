package config_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/9seconds/shebang/internal/config"
	v1 "github.com/9seconds/shebang/internal/config/v1"
	"github.com/9seconds/shebang/internal/utils"
	"github.com/stretchr/testify/suite"
)

type ParseTestSuite struct {
	suite.Suite
}

func (suite *ParseTestSuite) TestParse() {
	for _, test := range []struct {
		name string
		doc  string
		want v1.Config
	}{
		{
			name: "empty input",
			want: v1.Config{
				Argv: []string{"bash"},
			},
		},
		{
			name: "script without marker",
			doc:  "#!/bin/bash\n# execute \"sh\"\necho привет\n",
			want: v1.Config{
				Argv: []string{"bash"},
			},
		},
		{
			name: "version zero defaults",
			doc:  "#!shebang.0\n",
			want: v1.Config{
				Argv: []string{"bash"},
			},
		},
		{
			name: "version one defaults",
			doc:  "#!shebang.1\n",
			want: v1.Config{
				Argv: []string{"bash"},
			},
		},
		{
			name: "version zero uses v1 parser",
			doc: "#!shebang.0\n" +
				"# execute \"sh\" \"-eu\"\n" +
				"# description \"привет\"\n" +
				"# example \"script source\"\n",
			want: v1.Config{
				Argv:        []string{"sh", "-eu"},
				Description: "привет",
				Example:     "script source",
			},
		},
		{
			name: "version one uses v1 parser",
			doc: "#!shebang.1\n" +
				"# execute \"sh\" \"-eu\"\n" +
				"# description \"привет\"\n" +
				"# example \"script source\"\n",
			want: v1.Config{
				Argv:        []string{"sh", "-eu"},
				Description: "привет",
				Example:     "script source",
			},
		},
		{
			name: "embedded configuration",
			doc: `#!/usr/bin/env shebang

#!shebang.1
# execute "sh" "-eu"
# description "привет"
# flag "verbose" {
#   short "v"
# }
# option "output" {
#   value "str"
# }
# arg "source"
# vararg "items"
# arg "dest"
echo "$@"
# unknown
`,
			want: v1.Config{
				Argv:        []string{"sh", "-eu"},
				Description: "привет",
				Flags: []v1.Flag{
					{
						Name:  "verbose",
						Short: "v",
					},
				},
				Options: []v1.Option{
					{
						Name:       "output",
						Type:       "str",
						Properties: map[string][]any{},
					},
				},
				FirstArgs: []v1.Arg{
					{
						Name: "source",
					},
				},
				VarArgs: &v1.VarArg{
					Name: "items",
				},
				LastArgs: []v1.Arg{
					{
						Name: "dest",
					},
				},
			},
		},
	} {
		suite.Run(test.name, func() {
			conf, err := config.Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)
			suite.Require().IsType(&v1.Config{}, conf)

			parsed := conf.(*v1.Config)
			suite.Require().NotNil(parsed)
			suite.Equal(test.want.Argv, parsed.Argv)
			suite.Equal(test.want.Description, parsed.Description)
			suite.Equal(test.want.Example, parsed.Example)
			suite.Equal(test.want.Flags, parsed.Flags)
			suite.Equal(test.want.Options, parsed.Options)
			suite.Equal(test.want.FirstArgs, parsed.FirstArgs)
			suite.Equal(test.want.LastArgs, parsed.LastArgs)
			suite.Equal(test.want.VarArgs, parsed.VarArgs)
		})
	}
}

func (suite *ParseTestSuite) TestErrors() {
	for _, test := range []struct {
		name    string
		doc     string
		message string
		want    error
	}{
		{
			name:    "unsupported version",
			doc:     "#!shebang.2\n",
			message: "unknown config version 2",
			want:    config.ErrUnknownConfigVersion,
		},
		{
			name:    "multiple digit version",
			doc:     "#!shebang.12\n",
			message: "unknown config version 12",
			want:    config.ErrUnknownConfigVersion,
		},
		{
			name:    "version checked before KDL parsing",
			doc:     "#!shebang.2\n# option \"unfinished\" {\n",
			message: "unknown config version 2",
			want:    config.ErrUnknownConfigVersion,
		},
		{
			name:    "version zero invalid KDL",
			doc:     "#!shebang.0\n# option \"unfinished\" {\n",
			message: "cannot parse config as KDL:",
		},
		{
			name:    "version one invalid KDL",
			doc:     "#!shebang.1\n# option \"unfinished\" {\n",
			message: "cannot parse config as KDL:",
		},
		{
			name:    "unknown node",
			doc:     "#!shebang.1\n# unknown\n",
			message: "cannot process node unknown:",
			want:    v1.ErrUnknownNode,
		},
		{
			name: "empty execute",
			doc:  "#!shebang.1\n# execute\n",
			want: v1.ErrDefineExecute,
		},
		{
			name:    "missing argument",
			doc:     "#!shebang.1\n# description\n",
			message: "cannot process node description:",
			want:    utils.ErrEmpty,
		},
		{
			name:    "oversized config line",
			doc:     "#!shebang.1\n#" + strings.Repeat("x", 128*1024),
			message: "token too long",
		},
	} {
		suite.Run(test.name, func() {
			conf, err := config.Parse(strings.NewReader(test.doc))
			suite.Require().Error(err)
			suite.Nil(conf)

			if test.message != "" {
				suite.Require().ErrorContains(err, test.message)
			}

			if test.want != nil {
				suite.Require().ErrorIs(err, test.want)
			}
		})
	}
}

func (suite *ParseTestSuite) TestReadErrors() {
	want := errors.New("read failed")

	for _, test := range []struct {
		name   string
		prefix string
	}{
		{
			name: "before marker",
		},
		{
			name:   "after marker",
			prefix: "#!shebang.1\n",
		},
		{
			name:   "after config line",
			prefix: "#!shebang.1\n# description \"привет\"\n",
		},
		{
			name:   "reader error before version dispatch",
			prefix: "#!shebang.2\n",
		},
	} {
		suite.Run(test.name, func() {
			input := io.MultiReader(strings.NewReader(test.prefix), iotest.ErrReader(want))
			conf, err := config.Parse(input)

			suite.Require().ErrorIs(err, want)
			suite.Nil(conf)
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	suite.Run(t, &ParseTestSuite{})
}
