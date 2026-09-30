package config

import (
	"fmt"
	"io"

	v1 "github.com/9seconds/shebang/internal/config/v1"
)

type Config interface {
	ExecArgv() []string
}

func Parse(r io.Reader) (Config, error) {
	version, reader, err := NewReader(r)
	if err != nil {
		return nil, err
	}

	switch version {
	case 0, 1:
		return v1.Parse(reader)
	}

	return nil, fmt.Errorf("unknown config version %d", version)
}
