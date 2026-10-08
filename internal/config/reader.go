package config

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strconv"
)

var (
	regexpConfigStart = regexp.MustCompile(`(?i)\s*#!\s*shebang\.(\d+)\s*`)
	regexpConfigLine  = regexp.MustCompile(`^\s*#(.*?)$`)
)

// NewReader extracts the version and consecutive configuration comment lines
// following a shebang marker. Without a marker, it returns version zero and an
// empty buffer.
func NewReader(r io.Reader) (int, *bytes.Buffer, error) {
	scanner := bufio.NewScanner(r)
	buf := &bytes.Buffer{}
	version := 0

	for scanner.Scan() {
		matches := regexpConfigStart.FindSubmatch(scanner.Bytes())
		if len(matches) != 2 {
			continue
		}

		version, _ = strconv.Atoi(string(matches[1]))

		break
	}

	for scanner.Scan() {
		matches := regexpConfigLine.FindSubmatch(scanner.Bytes())
		if len(matches) != 2 {
			break
		}

		_, _ = buf.Write(matches[1])
		_ = buf.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return 0, nil, err
	}

	return version, buf, nil
}
