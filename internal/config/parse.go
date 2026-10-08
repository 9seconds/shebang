package config

import (
	"fmt"
	"io"
	"strconv"

	"github.com/9seconds/shebang/internal/cli"
	v1 "github.com/9seconds/shebang/internal/config/v1"
	"github.com/9seconds/shebang/internal/log"
)

type Config interface {
	Configure(*cli.Command) error
}

func Parse(r io.Reader) (Config, error) {
	version, reader, err := NewReader(r)
	if err != nil {
		return nil, err
	}

	log.PrintVal("Detected config version", strconv.Itoa(version))
	log.PrintVal("Config", reader.String())

	switch version {
	case 0, 1:
		return v1.Parse(reader)
	}

	return nil, fmt.Errorf("unknown config version %d", version)
}
