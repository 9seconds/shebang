package cli_test

import (
	"os"
	"testing"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/stretchr/testify/suite"
)

type FlagTestSuite struct {
	suite.Suite
}

func (suite *FlagTestSuite) TestSetEnv() {
	for _, test := range []struct {
		name      string
		value     bool
		short     string
		wantLong  string
		wantShort string
	}{
		{
			name:      "false preserves environment",
			short:     "v",
			wantLong:  "old",
			wantShort: "old",
		},
		{
			name:      "true with short name",
			value:     true,
			short:     "v",
			wantLong:  "true",
			wantShort: "true",
		},
		{
			name:      "true without short name",
			value:     true,
			wantLong:  "true",
			wantShort: "old",
		},
	} {
		suite.Run(test.name, func() {
			suite.T().Setenv("SHEBANG_FL_VERBOSE", "old")
			suite.T().Setenv("SHEBANG_FS_V", "old")
			flag := cli.Flag{
				Long:  "verbose",
				Short: test.short,
				Value: test.value,
			}
			flag.SetEnv()
			suite.Equal(test.wantLong, os.Getenv("SHEBANG_FL_VERBOSE"))
			suite.Equal(test.wantShort, os.Getenv("SHEBANG_FS_V"))
		})
	}
}

func TestFlag(t *testing.T) {
	suite.Run(t, &FlagTestSuite{})
}
