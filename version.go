package main

import (
	"fmt"
	"runtime/debug"
)

const defaultVersion = "dev"

var version = ""

func printVersion() {
	fmt.Println(getVersion()) //nolint:forbidigo // We explicitly want to print to stdout
}

func getVersion() string {
	return resolveVersion(version, debug.ReadBuildInfo)
}

func resolveVersion(
	configured string,
	readBuildInfo func() (*debug.BuildInfo, bool),
) string {
	if configured != "" && configured != defaultVersion {
		return configured
	}

	if info, ok := readBuildInfo(); ok &&
		info != nil &&
		info.Main.Version != "" &&
		info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return defaultVersion
}

func wantsVersion(args []string) bool {
	if len(args) != 1 {
		return false
	}

	switch args[0] {
	case "-V", "--version":
		return true
	}

	return false
}
