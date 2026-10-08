package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func die(format string, arg ...any) {
	args := append([]any{format}, arg...)

	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		die("usage: shebang <script> <arg>...")
	}

	scriptName, err := filepath.Abs(os.Args[1])
	if err != nil {
		die("cannot execute %s: %s", os.Args[1], err)
	}

	scriptName, err = exec.LookPath(scriptName)
	if err != nil {
		die("cannot execute %s: %s", os.Args[1], err)
	}

	fmt.Println(scriptName)
}
