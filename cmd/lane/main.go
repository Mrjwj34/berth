package main

import (
	"fmt"
	"os"

	"github.com/Mrjwj34/lane/internal/version"
)

var (
	appVersion = "0.1.0-dev"
	commit     = "none"
	date       = "unknown"
)

func main() {
	info := version.Current(appVersion, commit, date)
	if err := newRoot(info).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "lane: %v\n", err)
		os.Exit(1)
	}
}
