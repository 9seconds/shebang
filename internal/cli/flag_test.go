package cli_test

import (
	"testing"

	"github.com/9seconds/shebang/internal/cli"
	"github.com/stretchr/testify/assert"
)

func TestFlagString(t *testing.T) {
	for _, test := range []struct {
		name  string
		value bool
		want  string
	}{
		{name: "false", want: "false"},
		{name: "true", value: true, want: "true"},
	} {
		t.Run(test.name, func(t *testing.T) {
			flag := &cli.Flag{Name: "verbose", Value: test.value}
			assert.Equal(t, test.want, flag.String())
		})
	}
}
