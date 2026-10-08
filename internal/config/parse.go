package config

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/9seconds/shebang/internal/cli"
	v1 "github.com/9seconds/shebang/internal/config/v1"
	"github.com/9seconds/shebang/internal/log"
)

// ErrUnknownConfigVersion indicates an unsupported configuration version.
var ErrUnknownConfigVersion = errors.New("unknown config version")

// Config configures a script command from parsed configuration.
type Config interface {
	Configure(*cli.Command) error
}

// Parse extracts embedded configuration from r and parses its detected version.
//
//nolint:ireturn // Configuration versions share the Config interface.
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

	return nil, fmt.Errorf("%w %d", ErrUnknownConfigVersion, version)
}
