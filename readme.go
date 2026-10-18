package main

import (
	_ "embed"
	"fmt"

	"charm.land/glamour/v2"
	"github.com/9seconds/shebang/internal/log"
)

//go:embed README.md
var readmeContent string

func renderReadme() {
	out, err := glamour.RenderWithEnvironmentConfig(readmeContent)
	if err != nil {
		log.Dief("cannot render a readme: %s", err)
	}

	fmt.Println(out) //nolint:forbidigo // We explicitly want to print to stdout
}

func wantsReadme(args []string) bool {
	switch len(args) {
	case 0:
		return true
	case 1:
	default:
		return false
	}

	switch args[0] {
	case "-h", "--help":
		return true
	}

	return false
}
