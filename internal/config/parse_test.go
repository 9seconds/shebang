package config_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/9seconds/shebang/internal/config"
	v1 "github.com/9seconds/shebang/internal/config/v1"
	"github.com/stretchr/testify/suite"
)

type ParseTestSuite struct {
	suite.Suite
}

func (suite *ParseTestSuite) TestSupportedVersions() {
	for _, test := range []struct {
		name string
		doc  string
	}{
		{name: "empty input"},
		{name: "script without config", doc: "#!/bin/bash\necho hello\n"},
		{name: "version zero", doc: "#!shebang.0\n#execute \"bash\"\n"},
		{name: "version one", doc: "#!shebang.1\n#execute \"bash\"\n"},
		{name: "empty config", doc: "#!shebang.1\n"},
		{
			name: "script with config",
			doc:  "#!/bin/bash\n#!shebang.1\n#execute \"bash\" \"-e\"\n#description \"A script\"\necho hello\n",
		},
		{
			name: "ignore comments after script body",
			doc:  "#!shebang.1\n#execute \"bash\"\necho hello\n#unknown\n",
		},
	} {
		suite.Run(test.name, func() {
			conf, err := config.Parse(strings.NewReader(test.doc))
			suite.Require().NoError(err)
			suite.Require().NotNil(conf)
			suite.IsType(&v1.Config{}, conf)
		})
	}
}

func (suite *ParseTestSuite) TestUnsupportedVersions() {
	for _, test := range []struct {
		name string
		doc  string
		want string
	}{
		{name: "version two", doc: "#!shebang.2\n#execute \"bash\"\n", want: "unknown config version 2"},
		{name: "multiple digit version", doc: "#!shebang.12\n", want: "unknown config version 12"},
		{name: "version checked before KDL", doc: "#!shebang.2\n#execute \"unterminated\n", want: "unknown config version 2"},
	} {
		suite.Run(test.name, func() {
			conf, err := config.Parse(strings.NewReader(test.doc))
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.EqualError(err, test.want)
		})
	}
}

func (suite *ParseTestSuite) TestInvalidConfig() {
	for _, test := range []struct {
		name string
		doc  string
		want string
	}{
		{name: "malformed KDL", doc: "#!shebang.1\n#execute \"unterminated\n", want: "cannot parse config as KDL:"},
		{name: "unknown node", doc: "#!shebang.1\n#unknown\n", want: "cannot process node unknown: unknown node type"},
		{name: "invalid field type", doc: "#!shebang.1\n#execute 42\n", want: "cannot process node execute: unexpected value of type int64, expected string"},
		{name: "invalid version zero config", doc: "#!shebang.0\n#description\n", want: "cannot process node description: expected 1 argument, got 0"},
	} {
		suite.Run(test.name, func() {
			conf, err := config.Parse(strings.NewReader(test.doc))
			suite.Require().Error(err)
			suite.Nil(conf)
			suite.ErrorContains(err, test.want)
		})
	}
}

func (suite *ParseTestSuite) TestReadErrors() {
	want := errors.New("read failed")
	for _, test := range []struct {
		name   string
		prefix string
	}{
		{name: "before marker"},
		{name: "after marker", prefix: "#!shebang.1\n"},
		{name: "after config line", prefix: "#!shebang.1\n#execute \"bash\"\n"},
	} {
		suite.Run(test.name, func() {
			input := io.MultiReader(strings.NewReader(test.prefix), iotest.ErrReader(want))
			conf, err := config.Parse(input)
			suite.ErrorIs(err, want)
			suite.Nil(conf)
		})
	}
}

func (suite *ParseTestSuite) TestOversizedConfigLine() {
	conf, err := config.Parse(strings.NewReader("#!shebang.1\n#" + strings.Repeat("x", 128*1024)))
	suite.Require().Error(err)
	suite.ErrorContains(err, "token too long")
	suite.Nil(conf)
}

func TestParse(t *testing.T) {
	suite.Run(t, &ParseTestSuite{})
}
