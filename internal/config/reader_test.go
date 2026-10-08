package config_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/9seconds/shebang/internal/config"
	"github.com/stretchr/testify/suite"
)

type ReaderTestSuite struct {
	suite.Suite
}

func (suite *ReaderTestSuite) TestExtractConfig() {
	for _, test := range []struct {
		name    string
		doc     string
		version int
		want    string
	}{
		{
			name: "empty input",
		},
		{
			name: "no marker",
			doc: "#!/bin/bash\n# execute \"bash\"\necho hello\n",
		},
		{
			name: "marker without version",
			doc: "#!shebang\n# execute \"bash\"\n",
		},
		{
			name: "marker with invalid version",
			doc: "#!shebang.one\n# execute \"bash\"\n",
		},
		{
			name: "version zero",
			doc: "#!shebang.0\n",
			version: 0,
		},
		{
			name: "version one",
			doc: "#!shebang.1\n",
			version: 1,
		},
		{name: "multiple digit version", doc: "#!shebang.12\n", version: 12},
		{
			name: "case insensitive marker",
			doc: "#!ShEbAnG.1\n",
			version: 1,
		},
		{
			name: "marker whitespace",
			doc: "  #!  shebang.1  \n",
			version: 1,
		},
		{
			name: "marker at eof",
			doc: "#!shebang.1",
			version: 1,
		},
		{
			name:    "skip script preamble",
			doc:     "#!/bin/bash\n# a script\necho before\n#!shebang.1\n#execute \"bash\"\n",
			version: 1,
			want:    "execute \"bash\"\n",
		},
		{
			name:    "multiple config lines",
			doc:     "#!shebang.1\n# execute \"bash\"\n# description \"A script\"\n",
			version: 1,
			want:    " execute \"bash\"\n description \"A script\"\n",
		},
		{
			name:    "indented comments",
			doc:     "#!shebang.1\n  #execute \"bash\"\n\t#description \"A script\"\n",
			version: 1,
			want:    "execute \"bash\"\ndescription \"A script\"\n",
		},
		{
			name:    "empty comment line",
			doc:     "#!shebang.1\n#\n#execute \"bash\"\n",
			version: 1,
			want:    "\nexecute \"bash\"\n",
		},
		{
			name:    "preserve hashes and whitespace",
			doc:     "#!shebang.1\n# description \"#hello\"  \n",
			version: 1,
			want:    " description \"#hello\"  \n",
		},
		{
			name:    "stop at script body",
			doc:     "#!shebang.1\n#execute \"bash\"\necho hello\n#description \"ignored\"\n",
			version: 1,
			want:    "execute \"bash\"\n",
		},
		{
			name:    "stop at blank line",
			doc:     "#!shebang.1\n#execute \"bash\"\n\n#description \"ignored\"\n",
			version: 1,
			want:    "execute \"bash\"\n",
		},
		{
			name:    "config line at eof",
			doc:     "#!shebang.1\n#execute \"bash\"",
			version: 1,
			want:    "execute \"bash\"\n",
		},
		{
			name:    "windows line endings",
			doc:     "#!shebang.1\r\n#execute \"bash\"\r\n",
			version: 1,
			want:    "execute \"bash\"\n",
		},
	} {
		suite.Run(test.name, func() {
			version, reader, err := config.NewReader(strings.NewReader(test.doc))
			suite.Require().NoError(err)
			suite.Require().NotNil(reader)
			suite.Equal(test.version, version)

			data, err := io.ReadAll(reader)
			suite.Require().NoError(err)
			suite.Equal(test.want, string(data))
		})
	}
}

func (suite *ReaderTestSuite) TestReadErrors() {
	want := errors.New("read failed")
	for _, test := range []struct {
		name   string
		prefix string
	}{
		{
			name: "before marker",
		},
		{
			name: "after marker",
			prefix: "#!shebang.1\n",
		},
		{
			name: "after config line",
			prefix: "#!shebang.1\n#execute \"bash\"\n",
		},
	} {
		suite.Run(test.name, func() {
			input := io.MultiReader(strings.NewReader(test.prefix), iotest.ErrReader(want))
			version, reader, err := config.NewReader(input)

			suite.ErrorIs(err, want)
			suite.Zero(version)
			suite.Nil(reader)
		})
	}
}

func (suite *ReaderTestSuite) TestOversizedLines() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{
			name: "before marker",
			doc: strings.Repeat("x", 128*1024),
		},
		{
			name: "config line",
			doc: "#!shebang.1\n#" + strings.Repeat("x", 128*1024),
		},
	} {
		suite.Run(test.name, func() {
			version, reader, err := config.NewReader(strings.NewReader(test.doc))
			suite.Require().Error(err)
			suite.ErrorContains(err, "token too long")
			suite.Zero(version)
			suite.Nil(reader)
		})
	}
}

func TestReader(t *testing.T) {
	suite.Run(t, &ReaderTestSuite{})
}
