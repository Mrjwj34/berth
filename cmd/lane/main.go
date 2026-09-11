package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/Mrjwj34/lane/internal/version"
)

var (
	appVersion = "0.2.0-dev"
	commit     = "none"
	date       = "unknown"
)

func main() {
	info := version.Current(appVersion, commit, date)
	if err := newRoot(info).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "lane: %v\n", err)
		var code *exec.ExitError
		if errors.As(err, &code) && code.ExitCode() > 0 {
			os.Exit(code.ExitCode())
		}
		os.Exit(1)
	}
}
